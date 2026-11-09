package service

import (
	"encoding/json"
	"fmt"
	"github.com/bwmarrin/discordgo"
	"github.com/signintech/gopdf"
	"hkd.nam2507/model"
	"image"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"
)

const HKDTemplatePath2 = "templates/2.docx"
const PowerOfAttorneyFile = "templates/uyquyen_dktlhkd.docx"

var NganhNgheDB []model.NganhNgheKinhDoanh

func init() {
	if envData, err := os.ReadFile(".env"); err == nil {
		lines := strings.Split(string(envData), "\n")
		for _, line := range lines {
			line = strings.TrimSpace(line)
			if strings.HasPrefix(line, "#") || !strings.Contains(line, "=") {
				continue
			}
			parts := strings.SplitN(line, "=", 2)
			key := strings.TrimSpace(parts[0])
			val := strings.TrimSpace(parts[1])
			if os.Getenv(key) == "" {
				os.Setenv(key, val)
			}
		}
	}
	Token = os.Getenv("DISCORD_BOT_TOKEN")

	data, err := os.ReadFile("nganh_nghe.json")
	if err != nil {
		data, err = os.ReadFile("../nganh_nghe.json")
	}
	if err != nil {
		panic("Không thể đọc file nganh_nghe.json: " + err.Error())
	}
	if err := json.Unmarshal(data, &NganhNgheDB); err != nil {
		panic("Không thể parse nganh_nghe.json: " + err.Error())
	}
}

var Token string

var (
	pendingThayDoiSessions = make(map[string]model.ThayDoiHKD)
	pendingThayDoiMu       sync.Mutex
)

func getInteractionUserID(i *discordgo.InteractionCreate) string {
	if i.Member != nil && i.Member.User != nil {
		return i.Member.User.ID
	}
	if i.User != nil {
		return i.User.ID
	}
	return ""
}

// ==================== ENTRY POINT ====================

func MessageCreate(s *discordgo.Session, m *discordgo.MessageCreate) {
	if m.Author.Bot {
		return
	}
	switch {
	case strings.HasPrefix(m.Content, "!register_hkd"):
		handleHKDRegistration(s, m)
	case strings.HasPrefix(m.Content, "!thaydoi_hkd") || strings.HasPrefix(m.Content, "!thaydoi"):
		handlePromptThayDoiMenu(s, m)
	case strings.HasPrefix(m.Content, "!submit_thaydoi"):
		handleSubmitThayDoi(s, m)
	case strings.HasPrefix(m.Content, "!help"):
		sendHelpMessage(s, m.ChannelID)
	case strings.HasPrefix(m.Content, "!ma"):
		listNganhNghe(s, m.ChannelID)
	default:
		// Kiểm tra nếu user đang chờ gửi ảnh
		handlePendingImages(s, m)
	}
}

// ==================== HANDLERS ====================

var pendingImageSessions = map[string]model.Hokinhdoanh{}
var pendingMu sync.Mutex

func handleHKDRegistration(s *discordgo.Session, m *discordgo.MessageCreate) {
	sendErr := func(msg string) { s.ChannelMessageSend(m.ChannelID, msg) }

	fields := parseFields(strings.TrimPrefix(m.Content, "!register_hkd"))

	if errs := validateRequiredFields(fields); len(errs) > 0 {
		sendErr("❌ Lỗi thông tin:\n" + strings.Join(errs, "\n"))
		return
	}

	nganhNghe := searchNganhNgheByMaNganh(get(fields, "Ngành nghề kinh doanh"))
	if len(nganhNghe) == 0 {
		sendErr("❌ Ngành nghề kinh doanh không hợp lệ!")
		return
	}

	sample := buildHoKinhDoanhModel(fields, nganhNghe)

	if err := fillHKDTemplate(sample, HKDTemplatePath2); err != nil {
		sendErr("❌ Lỗi tạo tài liệu: " + err.Error())
		return
	}

	if err := generatePowerOfAttorney(sample, PowerOfAttorneyFile); err != nil {
		sendErr("❌ Lỗi tạo giấy ủy quyền: " + err.Error())
		return
	}

	// Lưu session chờ ảnh CCCD
	pendingMu.Lock()
	pendingImageSessions[m.Author.ID] = sample
	pendingMu.Unlock()

	var nganhNgheList strings.Builder
	for i, nn := range sample.NganhNgheKinhDoanh {
		nganhNgheList.WriteString(fmt.Sprintf("%d. **%d** - %s\n", i+1, nn.MaNganh, strings.TrimSpace(nn.TenNganh)))
	}

	s.ChannelMessageSend(m.ChannelID, fmt.Sprintf(
		"✅ Tạo hồ sơ thành công cho **%s**!\n\n📋 **Ngành nghề đăng ký:**\n%s\n📎 Vui lòng gửi 2 ảnh CCCD để hoàn tất.",
		sample.HoVaTen,
		nganhNgheList.String(),
	))
}

func handlePendingImages(s *discordgo.Session, m *discordgo.MessageCreate) {
	if len(m.Attachments) == 0 {
		return
	}

	pendingMu.Lock()
	sample, exists := pendingImageSessions[m.Author.ID]
	pendingMu.Unlock()

	if !exists {
		return
	}

	if len(m.Attachments) < 2 {
		s.ChannelMessageSend(m.ChannelID, "⚠️ Vui lòng gửi đủ 2 ảnh CCCD!")
		return
	}

	sendErr := func(msg string) { s.ChannelMessageSend(m.ChannelID, msg) }

	// Download ảnh
	var imagePaths []string
	for i, att := range m.Attachments[:2] {
		resp, err := http.Get(att.URL)
		if err != nil {
			sendErr("❌ Không thể tải ảnh: " + err.Error())
			return
		}
		defer resp.Body.Close()

		tmpFile := fmt.Sprintf("temp_img_%s_%d.jpg", m.Author.ID, i)
		f, _ := os.Create(tmpFile)
		io.Copy(f, resp.Body)
		f.Close()
		imagePaths = append(imagePaths, tmpFile)
		defer os.Remove(tmpFile)
	}

	// Lưu PDF vào cùng thư mục hồ sơ
	folderName := sanitizeFolderName(sample.HoVaTen)
	outputDir := filepath.Join(getOutputPath(), folderName)
	pdfPath := filepath.Join(outputDir, "BẢN SAO GIẤY TỜ PHÁP LÝ CỦA CÁ NHÂN_"+strings.ToUpper(sample.HoVaTen)+"_0001.pdf")

	if err := imagesToPDF(imagePaths, pdfPath); err != nil {
		sendErr("❌ Lỗi tạo PDF CCCD: " + err.Error())
		return
	}

	// Xóa session và tin nhắn chứa ảnh
	pendingMu.Lock()
	delete(pendingImageSessions, m.Author.ID)
	pendingMu.Unlock()

	// Timeout 1 phút nếu không gửi ảnh
	go func(userID string) {
		time.Sleep(1 * time.Minute)
		pendingMu.Lock()
		if _, exists := pendingImageSessions[userID]; exists {
			delete(pendingImageSessions, userID)
			pendingMu.Unlock()
			s.ChannelMessageSend(m.ChannelID, "⏰ Hết thời gian chờ ảnh CCCD cho **"+sample.HoVaTen+"**! Vui lòng tạo lại hồ sơ.")
		} else {
			pendingMu.Unlock()
		}
	}(m.Author.ID)

	s.ChannelMessageDelete(m.ChannelID, m.ID)
	s.ChannelMessageSend(m.ChannelID, "✅ Đã lưu ảnh CCCD cho "+sample.HoVaTen+"!")
}

func imagesToPDF(imagePaths []string, outputPath string) error {
	pdf := gopdf.GoPdf{}
	pdf.Start(gopdf.Config{PageSize: *gopdf.PageSizeA4})

	pageW := 595.0
	pageH := 842.0
	maxW := pageW * 0.5
	maxH := pageH * 0.5

	for _, imgPath := range imagePaths {
		// Đọc kích thước thực của ảnh
		f, err := os.Open(imgPath)
		if err != nil {
			return err
		}
		imgConfig, _, err := image.DecodeConfig(f)
		f.Close()
		if err != nil {
			return err
		}

		// Tính tỉ lệ để fit trong maxW x maxH mà không bị méo
		imgW := float64(imgConfig.Width)
		imgH := float64(imgConfig.Height)
		ratio := min(maxW/imgW, maxH/imgH)

		finalW := imgW * ratio
		finalH := imgH * ratio

		// Căn giữa trang
		x := (pageW - finalW) / 2
		y := (pageH - finalH) / 2

		pdf.AddPage()
		if err := pdf.Image(imgPath, x, y, &gopdf.Rect{W: finalW, H: finalH}); err != nil {
			return fmt.Errorf("lỗi thêm ảnh %s: %w", imgPath, err)
		}
	}

	return pdf.WritePdf(outputPath)
}

func min(a, b float64) float64 {
	if a < b {
		return a
	}
	return b
}

// ==================== HELPERS ====================

func validateRequiredFields(fields map[string]string) []string {
	var errors []string

	required := []string{"Họ và tên", "Địa chỉ thường trú", "Ngày sinh", "Ngày cấp CCCD"}
	for _, field := range required {
		if get(fields, field) == "" {
			errors = append(errors, fmt.Sprintf("• %s không được để trống", field))
		}
	}

	if cccd := get(fields, "CCCD"); len(strings.TrimSpace(cccd)) != 12 {
		errors = append(errors, "• CCCD phải có đúng 12 chữ số")
	}

	if sdt := get(fields, "Số điện thoại"); len(strings.TrimSpace(sdt)) != 10 {
		errors = append(errors, "• Số điện thoại phải có đúng 10 chữ số")
	}

	return errors
}

func buildHoKinhDoanhModel(fields map[string]string, nganhNghe []model.NganhNgheKinhDoanh) model.Hokinhdoanh {
	fullName := get(fields, "Họ và tên")
	diaChiThuongTru := get(fields, "Địa chỉ thường trú")
	diaChiKinhDoanh := get(fields, "Địa chỉ kinh doanh")

	// Nếu thường trú không phải HCM thì liên lạc = kinh doanh
	diaChiLienLac := get(fields, "Địa chỉ liên lạc")
	if diaChiLienLac == "" {
		if parseAddress(diaChiThuongTru).TinhTP != "Thành phố Hồ Chí Minh" {
			diaChiLienLac = diaChiKinhDoanh
		} else {
			diaChiLienLac = diaChiThuongTru
		}
	}

	return model.Hokinhdoanh{
		HoVaTen:        fullName,
		GioiTinh:       get(fields, "Giới tính", "Nam"),
		NgaySinh:       get(fields, "Ngày sinh"),
		CCCD:           get(fields, "CCCD"),
		CoQuan:         get(fields, "Cơ quan", parseAddress(diaChiKinhDoanh).XaPhuong),
		CoQuanCap:      get(fields, "Nơi cấp CCCD", "Cục Trưởng Cục Cảnh Sát Quản Lý Hành Chính Về Trật Tự Xã Hội"),
		NgayCap:        get(fields, "Ngày cấp CCCD"),
		DanToc:         get(fields, "Dân tộc", "Kinh"),
		SDT:            get(fields, "Số điện thoại"),
		TenHoKinhDoanh: get(fields, "Tên hộ kinh doanh", fullName),
		VonKinhDoanh: model.VonKinhDoanh{
			BangSo:  get(fields, "Vốn kinh doanh (Bằng Số)", "30.000.000"),
			BangChu: get(fields, "Vốn kinh doanh (Bằng Chữ)", "Ba mươi"),
		},
		DiaChiThuongTru:    parseAddress(diaChiThuongTru),
		DiaChiLienLac:      parseAddress(diaChiLienLac),
		DiaChiKinhDoanh:    parseAddress(diaChiKinhDoanh),
		NganhNgheKinhDoanh: nganhNghe,
	}
}

func listNganhNghe(s *discordgo.Session, channelID string) {
	var msg strings.Builder
	msg.WriteString("**📋 Danh sách Ngành nghề Kinh doanh có sẵn:**\n\n")
	for _, nn := range NganhNgheDB {
		msg.WriteString(fmt.Sprintf("**%d** - %s\n", nn.MaNganh, nn.TenNganh))
	}
	msg.WriteString("\n*Sử dụng mã số trong lệnh đăng ký*")
	s.ChannelMessageSend(channelID, msg.String())
}

func sendHelpMessage(s *discordgo.Session, channelID string) {
	helpMsg := `**🏢 Hướng dẫn sử dụng Bot Quản lý Hộ Kinh Doanh**

**1. Tạo mới hộ kinh doanh:**
` + "`!register_hkd`" + ` - Điền thông tin tạo giấy đề nghị đăng ký thành lập HKD

**2. Thay đổi nội dung đăng ký hộ kinh doanh:**
` + "`!thaydoi_hkd`" + ` - Hiện danh sách mục thay đổi (chọn bằng tick ✓)
` + "`!submit_thaydoi`" + ` - Gửi thông tin thay đổi để tạo văn bản

**3. Tra cứu:**
- ` + "`!ma`" + ` - Xem danh sách mã ngành nghề
- ` + "`!help`" + ` - Xem hướng dẫn này`

	s.ChannelMessageSend(channelID, helpMsg)
}

// ==================== THAY ĐỔI HKD HANDLERS ====================

func getNganhNgheSelectOptions() []discordgo.SelectMenuOption {
	var opts []discordgo.SelectMenuOption
	seen := make(map[string]bool)

	for _, nn := range NganhNgheDB {
		val := strconv.Itoa(nn.MaNganh)
		if seen[val] {
			continue // Tránh duplicate mã ngành làm lỗi Discord 400
		}
		seen[val] = true

		rawName := strings.Split(nn.TenNganh, "\n")[0]
		rawName = strings.TrimSpace(rawName)
		if len(rawName) > 80 {
			rawName = rawName[:77] + "..."
		}
		label := fmt.Sprintf("%d - %s", nn.MaNganh, rawName)
		if len(label) > 100 {
			label = label[:97] + "..."
		}

		desc := ""
		if strings.Contains(nn.TenNganh, "Chi tiết:") {
			sp := strings.Split(nn.TenNganh, "Chi tiết:")
			if len(sp) > 1 {
				desc = strings.TrimSpace(strings.ReplaceAll(sp[1], ")", ""))
				desc = strings.ReplaceAll(desc, "\n", " ")
				if len(desc) > 100 {
					desc = desc[:97] + "..."
				}
			}
		}

		opts = append(opts, discordgo.SelectMenuOption{
			Label:       label,
			Value:       val,
			Description: desc,
		})

		if len(opts) >= 25 {
			break
		}
	}
	return opts
}

func thayDoiMenuOptions() []discordgo.SelectMenuOption {
	return []discordgo.SelectMenuOption{
		{
			Label:       "1. Thay đổi tên Hộ kinh doanh",
			Value:       "ten",
			Description: "Đổi tên tiếng Việt của HKD",
			Emoji:       &discordgo.ComponentEmoji{Name: "🏷️"},
		},
		{
			Label:       "2. Thay đổi địa chỉ trụ sở",
			Value:       "truso",
			Description: "Đổi số nhà, đường, phường, SĐT...",
			Emoji:       &discordgo.ComponentEmoji{Name: "🏠"},
		},
		{
			Label:       "3. Thay đổi vốn kinh doanh",
			Value:       "von",
			Description: "Vốn mới, thời gian, hình thức tăng/giảm",
			Emoji:       &discordgo.ComponentEmoji{Name: "💰"},
		},
		{
			Label:       "4. BỔ SUNG ngành nghề kinh doanh",
			Value:       "bosung_nganh",
			Description: "Tick chọn nhiều ngành nghề để bổ sung thêm",
			Emoji:       &discordgo.ComponentEmoji{Name: "➕"},
		},
		{
			Label:       "5. BỎ BỚT ngành nghề kinh doanh",
			Value:       "bo_nganh",
			Description: "Tick chọn nhiều ngành nghề để bỏ bớt khỏi danh sách",
			Emoji:       &discordgo.ComponentEmoji{Name: "➖"},
		},
		{
			Label:       "6. VỪA BỔ SUNG VỪA BỎ BỚT ngành nghề",
			Value:       "both_nganh",
			Description: "Chọn cả ngành nghề thêm vào và ngành nghề bỏ bớt",
			Emoji:       &discordgo.ComponentEmoji{Name: "🔄"},
		},
		{
			Label:       "7. Thay đổi số lao động",
			Value:       "laodong",
			Description: "Cập nhật tổng số lao động (thuế)",
			Emoji:       &discordgo.ComponentEmoji{Name: "👥"},
		},
		{
			Label:       "8. Đăng ký cấp đổi sang GCN đăng ký HKD",
			Value:       "capdoi",
			Description: "Tick vào ô Cá nhân và điền thông tin cấp đổi",
			Emoji:       &discordgo.ComponentEmoji{Name: "📑"},
		},
	}
}

func handlePromptThayDoiMenu(s *discordgo.Session, m *discordgo.MessageCreate) {
	minVals := 1
	msg := &discordgo.MessageSend{
		Content: "📝 **THAY ĐỔI NỘI DUNG ĐĂNG KÝ HỘ KINH DOANH**\n\n" +
			"👉 Bạn có thể bấm nút **Nhập nhanh mã ngành** hoặc **tick chọn menu** bên dưới:",
		Components: []discordgo.MessageComponent{
			discordgo.ActionsRow{
				Components: []discordgo.MessageComponent{
					discordgo.Button{
						Label:    "⌨️ Nhập nhanh mã ngành & thông tin",
						Style:    discordgo.PrimaryButton,
						CustomID: "btn_quick_thaydoi_all",
						Emoji:    &discordgo.ComponentEmoji{Name: "⚡"},
					},
				},
			},
			discordgo.ActionsRow{
				Components: []discordgo.MessageComponent{
					discordgo.SelectMenu{
						CustomID:    "select_thaydoi_items",
						Placeholder: "Hoặc tick chọn các mục bạn muốn thay đổi...",
						MinValues:   &minVals,
						MaxValues:   8,
						Options:     thayDoiMenuOptions(),
					},
				},
			},
		},
	}

	_, err := s.ChannelMessageSendComplex(m.ChannelID, msg)
	if err != nil {
		s.ChannelMessageSend(m.ChannelID, "❌ Lỗi hiển thị menu: "+err.Error())
	}
}

func RegisterSlashCommands(dg *discordgo.Session) {
	commands := []*discordgo.ApplicationCommand{
		{
			Name:        "thaydoi_hkd",
			Description: "Thay đổi nội dung đăng ký hộ kinh doanh (có menu tick chọn & nhập nhanh)",
		},
		{
			Name:        "help",
			Description: "Xem hướng dẫn sử dụng bot",
		},
		{
			Name:        "ma",
			Description: "Xem danh sách mã ngành nghề kinh doanh",
		},
	}

	appID := dg.State.User.ID
	for _, cmd := range commands {
		_, err := dg.ApplicationCommandCreate(appID, "", cmd)
		if err != nil {
			fmt.Printf("Lỗi đăng ký global slash command %s: %v\n", cmd.Name, err)
		}
	}

	// Đăng ký trực tiếp cho từng server (guild) để hiện ngay lập tức
	for _, g := range dg.State.Guilds {
		for _, cmd := range commands {
			_, err := dg.ApplicationCommandCreate(appID, g.ID, cmd)
			if err != nil {
				fmt.Printf("Lỗi đăng ký guild command %s in %s: %v\n", cmd.Name, g.ID, err)
			}
		}
	}
	fmt.Println("✅ Đã đăng ký Slash Commands (/) thành công.")
}

func handleSlashThayDoi(s *discordgo.Session, i *discordgo.InteractionCreate) {
	minVals := 1
	err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: "📝 **THAY ĐỔI NỘI DUNG ĐĂNG KÝ HỘ KINH DOANH**\n\n" +
				"👉 Bạn có thể bấm nút **Nhập nhanh mã ngành** hoặc **tick chọn menu** bên dưới:",
			Components: []discordgo.MessageComponent{
				discordgo.ActionsRow{
					Components: []discordgo.MessageComponent{
						discordgo.Button{
							Label:    "⌨️ Nhập nhanh mã ngành & thông tin",
							Style:    discordgo.PrimaryButton,
							CustomID: "btn_quick_thaydoi_all",
							Emoji:    &discordgo.ComponentEmoji{Name: "⚡"},
						},
					},
				},
				discordgo.ActionsRow{
					Components: []discordgo.MessageComponent{
						discordgo.SelectMenu{
							CustomID:    "select_thaydoi_items",
							Placeholder: "Hoặc tick chọn các mục bạn muốn thay đổi...",
							MinValues:   &minVals,
							MaxValues:   8,
							Options:     thayDoiMenuOptions(),
						},
					},
				},
			},
		},
	})
	if err != nil {
		fmt.Printf("Lỗi slash thaydoi: %v\n", err)
	}
}

func handleSlashHelp(s *discordgo.Session, i *discordgo.InteractionCreate) {
	helpMsg := `**🏢 Hướng dẫn sử dụng Bot Quản lý Hộ Kinh Doanh**

**1. Tạo mới hộ kinh doanh:**
` + "`!register_hkd`" + ` - Điền thông tin tạo giấy đề nghị đăng ký thành lập HKD

**2. Thay đổi nội dung đăng ký hộ kinh doanh:**
- Gõ ` + "`/thaydoi_hkd`" + ` hoặc ` + "`!thaydoi_hkd`" + ` - Hiện menu tick chọn các mục cần thay đổi
- Gõ ` + "`!submit_thaydoi`" + ` - Gửi thông tin thay đổi

**3. Tra cứu:**
- ` + "`/ma`" + ` hoặc ` + "`!ma`" + ` - Xem danh sách mã ngành nghề
- ` + "`/help`" + ` hoặc ` + "`!help`" + ` - Xem hướng dẫn này`

	_ = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: helpMsg,
		},
	})
}

func handleSlashMa(s *discordgo.Session, i *discordgo.InteractionCreate) {
	var msg strings.Builder
	msg.WriteString("**📋 Danh sách Ngành nghề Kinh doanh có sẵn:**\n\n")
	for _, nn := range NganhNgheDB {
		msg.WriteString(fmt.Sprintf("**%d** - %s\n", nn.MaNganh, nn.TenNganh))
	}
	msg.WriteString("\n*Sử dụng mã số trong lệnh đăng ký / thay đổi*")

	_ = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: msg.String(),
		},
	})
}

func InteractionCreate(s *discordgo.Session, i *discordgo.InteractionCreate) {
	switch i.Type {
	case discordgo.InteractionApplicationCommand:
		data := i.ApplicationCommandData()
		switch data.Name {
		case "thaydoi_hkd":
			handleSlashThayDoi(s, i)
		case "help":
			handleSlashHelp(s, i)
		case "ma":
			handleSlashMa(s, i)
		}
	case discordgo.InteractionMessageComponent:
		data := i.MessageComponentData()

		// 0. Xử lý các nút bấm nhập nhanh (Buttons)
		if data.CustomID == "btn_quick_thaydoi_all" {
			openModalForThayDoi(s, i, "modal_thaydoi_submit:quick:0", "Mã ngành thay đổi HOẶC Nội dung thay đổi", "VD: 1071 (chính), 4711 HOẶC Bổ sung: 1071, Bỏ: 5610", "")
			return
		}

		if strings.HasPrefix(data.CustomID, "btn_open_modal_bs:") {
			capDoiFlag := strings.TrimPrefix(data.CustomID, "btn_open_modal_bs:")
			openModalForThayDoi(s, i, "modal_thaydoi_submit:bs=:bo=:"+capDoiFlag, "Mã ngành BỔ SUNG (kèm chi tiết & chính)", "VD: 1071 (chính), 4711 HOẶC 1071: Sản xuất bánh ướt (chính)", "")
			return
		}

		if strings.HasPrefix(data.CustomID, "btn_open_modal_bo:") {
			capDoiFlag := strings.TrimPrefix(data.CustomID, "btn_open_modal_bo:")
			openModalForThayDoi(s, i, "modal_thaydoi_submit:bs=:bo=:"+capDoiFlag, "Mã ngành BỎ BỚT", "VD: 5610, 4711", "")
			return
		}

		if strings.HasPrefix(data.CustomID, "btn_open_modal_both:") {
			capDoiFlag := strings.TrimPrefix(data.CustomID, "btn_open_modal_both:")
			defaultPrompt := "=== BỔ SUNG (STT | Mã | Chi tiết | (chính)) ===\n1 |  |  | (chính)\n\n=== BỎ BỚT (STT | Mã | Tên ngành) ===\n1 |  | "
			openModalForThayDoi(s, i, "modal_thaydoi_submit:both:0:"+capDoiFlag, "Mã ngành BỔ SUNG & BỎ (mỗi dòng 1 loại)", "Bổ sung trước, bỏ bớt sau", defaultPrompt)
			return
		}

		// Nút bấm hỏi thêm Thông tin thêm ({thongtinthem})
		if strings.HasPrefix(data.CustomID, "btn_prompt_thongtinthem:") {
			targetUserID := strings.TrimPrefix(data.CustomID, "btn_prompt_thongtinthem:")
			_ = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
				Type: discordgo.InteractionResponseModal,
				Data: &discordgo.InteractionResponseData{
					CustomID: "modal_thongtinthem_submit:" + targetUserID,
					Title:    "Thông tin thêm địa chỉ mới",
					Components: []discordgo.MessageComponent{
						discordgo.ActionsRow{
							Components: []discordgo.MessageComponent{
								discordgo.TextInput{
									CustomID:    "thong_tin_them",
									Label:       "Thông tin thêm ({thongtinthem})",
									Style:       discordgo.TextInputParagraph,
									Placeholder: "VD: Tầng 3, phòng 302, Tòa nhà Bitexco...",
									Required:    false,
								},
							},
						},
					},
				},
			})
			return
		}

		if strings.HasPrefix(data.CustomID, "btn_skip_thongtinthem:") {
			targetUserID := strings.TrimPrefix(data.CustomID, "btn_skip_thongtinthem:")
			pendingThayDoiMu.Lock()
			pendingData, exists := pendingThayDoiSessions[targetUserID]
			pendingThayDoiMu.Unlock()

			if exists {
				_ = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
					Type: discordgo.InteractionResponseChannelMessageWithSource,
					Data: &discordgo.InteractionResponseData{
						Content: "⏳ **Đang tiếp tục hoàn tất hồ sơ...**",
					},
				})
				checkAndProceedThayDoi(s, i.ChannelID, targetUserID, pendingData)
			}
			return
		}

		// Nút bấm hỏi thêm Ô số 6 Số lao động (khi Cấp đổi GCN)
		if strings.HasPrefix(data.CustomID, "btn_prompt_solaodong:") {
			targetUserID := strings.TrimPrefix(data.CustomID, "btn_prompt_solaodong:")
			_ = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
				Type: discordgo.InteractionResponseModal,
				Data: &discordgo.InteractionResponseData{
					CustomID: "modal_solaodong_submit:" + targetUserID,
					Title:    "Số lao động (Ô số 6)",
					Components: []discordgo.MessageComponent{
						discordgo.ActionsRow{
							Components: []discordgo.MessageComponent{
								discordgo.TextInput{
									CustomID:    "so_lao_dong",
									Label:       "Tổng số lao động (Mục 5.2)",
									Style:       discordgo.TextInputShort,
									Placeholder: "VD: 3",
									Required:    false,
								},
							},
						},
					},
				},
			})
			return
		}

		if strings.HasPrefix(data.CustomID, "btn_skip_solaodong:") {
			targetUserID := strings.TrimPrefix(data.CustomID, "btn_skip_solaodong:")
			pendingThayDoiMu.Lock()
			pendingData, exists := pendingThayDoiSessions[targetUserID]
			delete(pendingThayDoiSessions, targetUserID)
			pendingThayDoiMu.Unlock()

			if exists {
				_ = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
					Type: discordgo.InteractionResponseChannelMessageWithSource,
					Data: &discordgo.InteractionResponseData{
						Content: "⏳ **Đang tạo văn bản Word...**",
					},
				})
				pendingData.SoLaoDong = ""
				generateAndSendThayDoiDocx(s, i.ChannelID, pendingData)
			}
			return
		}

		// 1. Trường hợp Bước 1 khi chọn cả Bổ sung & Bỏ: người dùng đã chọn xong các mã Bổ sung
		if strings.HasPrefix(data.CustomID, "select_nganh_step1_bs:") {
			capDoiFlag := strings.TrimPrefix(data.CustomID, "select_nganh_step1_bs:")
			bsCodes := strings.Join(data.Values, ",")

			opts := getNganhNgheSelectOptions()
			minV := 1
			maxV := len(opts)
			if maxV > 15 {
				maxV = 15
			}

			err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
				Type: discordgo.InteractionResponseChannelMessageWithSource,
				Data: &discordgo.InteractionResponseData{
					Content: fmt.Sprintf("✅ **Đã chọn các ngành BỔ SUNG:** `%s`\n\n"+
						"📋 **BƯỚC 2/2: CHỌN CÁC NGÀNH NGHỀ BỎ BỚT**\n"+
						"Vui lòng tick chọn **các mã ngành BỎ BỚT** từ Drop list bên dưới (chọn được nhiều mã cùng lúc):\n"+
						"*(Chọn xong bảng điền thông tin sẽ tự động mở)*", bsCodes),
					Components: []discordgo.MessageComponent{
						discordgo.ActionsRow{
							Components: []discordgo.MessageComponent{
								discordgo.SelectMenu{
									CustomID:    fmt.Sprintf("select_nganh_step2_bo:%s:%s", capDoiFlag, bsCodes),
									Placeholder: "Bấm vào đây để tick chọn các ngành BỎ BỚT...",
									MinValues:   &minV,
									MaxValues:   maxV,
									Options:     opts,
								},
							},
						},
					},
				},
			})
			if err != nil {
				fmt.Printf("Lỗi gửi select menu bỏ step 2: %v\n", err)
			}
			return
		}

		// 2. Trường hợp Bước 2: người dùng đã chọn xong các mã Bỏ bớt -> Mở Modal
		if strings.HasPrefix(data.CustomID, "select_nganh_step2_bo:") {
			parts := strings.Split(data.CustomID, ":")
			capDoiFlag := "0"
			bsCodes := ""
			if len(parts) >= 2 {
				capDoiFlag = parts[1]
			}
			if len(parts) >= 3 {
				bsCodes = parts[2]
			}
			boCodes := strings.Join(data.Values, ",")

			modalCustomID := fmt.Sprintf("modal_thaydoi_submit:bs=%s:bo=%s:%s", bsCodes, boCodes, capDoiFlag)
			var bsList []string
			if bsCodes != "" {
				bsList = strings.Split(bsCodes, ",")
			}
			defaultVal := formatBothNganhDefaultValue(bsList, data.Values)
			openModalForThayDoi(s, i, modalCustomID, "Mã ngành BỔ SUNG & BỎ BỚT", "Bổ sung trước, bỏ bớt sau", defaultVal)
			return
		}

		// 3. Trường hợp chỉ chọn BỔ SUNG ngành nghề -> Mở Modal
		if strings.HasPrefix(data.CustomID, "select_nganh_bs_only:") {
			capDoiFlag := strings.TrimPrefix(data.CustomID, "select_nganh_bs_only:")
			bsCodes := strings.Join(data.Values, ",")

			modalCustomID := fmt.Sprintf("modal_thaydoi_submit:bs=%s:bo=:%s", bsCodes, capDoiFlag)
			defaultVal := formatBoSungDefaultValue(data.Values)
			openModalForThayDoi(s, i, modalCustomID, "Ngành BỔ SUNG (kèm chi tiết & chính)", "STT | Mã ngành | (sau đó mới điền chi tiết) | Tick box xem có phải ngành chính hay không", defaultVal)
			return
		}

		// 4. Trường hợp chỉ chọn BỎ BỚT ngành nghề -> Mở Modal
		if strings.HasPrefix(data.CustomID, "select_nganh_bo_only:") {
			capDoiFlag := strings.TrimPrefix(data.CustomID, "select_nganh_bo_only:")
			boCodes := strings.Join(data.Values, ",")

			modalCustomID := fmt.Sprintf("modal_thaydoi_submit:bs=:bo=%s:%s", boCodes, capDoiFlag)
			defaultVal := formatBoDefaultValue(data.Values)
			openModalForThayDoi(s, i, modalCustomID, "Ngành BỎ BỚT (STT | Mã | Tên)", "STT | Mã ngành | Tên ngành", defaultVal)
			return
		}

		// 5. Fallback tương thích ngược
		if strings.HasPrefix(data.CustomID, "select_nganh_nghe_item:") {
			selectedMa := data.Values[0]
			hasCapDoi := strings.HasSuffix(data.CustomID, ":1")
			capDoiStr := "0"
			if hasCapDoi {
				capDoiStr = "1"
			}
			modalCustomID := fmt.Sprintf("modal_thaydoi_submit:bs=%s:bo=:%s", selectedMa, capDoiStr)
			openModalForThayDoi(s, i, modalCustomID, "Mã ngành BỔ SUNG", "VD: 1071 (chính)", selectedMa)
			return
		}

		if data.CustomID != "select_thaydoi_items" {
			return
		}

		selected := data.Values
		hasTen := false
		hasTruSo := false
		hasVon := false
		hasBoSung := false
		hasBo := false
		hasLaoDong := false
		hasCapDoi := false

		for _, v := range selected {
			switch v {
			case "ten":
				hasTen = true
			case "truso":
				hasTruSo = true
			case "von":
				hasVon = true
			case "bosung_nganh", "nganhnghe":
				hasBoSung = true
			case "bo_nganh":
				hasBo = true
			case "both_nganh":
				hasBoSung = true
				hasBo = true
			case "laodong":
				hasLaoDong = true
			case "capdoi":
				hasCapDoi = true
			}
		}

		capDoiFlag := "0"
		if hasCapDoi {
			capDoiFlag = "1"
		}

		opts := getNganhNgheSelectOptions()
		minV := 1
		maxV := len(opts)
		if maxV > 15 {
			maxV = 15
		}

		// Nhánh 1: Vừa BỔ SUNG vừa BỎ BỚT ngành nghề
		if hasBoSung && hasBo {
			err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
				Type: discordgo.InteractionResponseChannelMessageWithSource,
				Data: &discordgo.InteractionResponseData{
					Content: "📋 **VỪA BỔ SUNG VỪA BỎ BỚT NGÀNH NGHỀ**\n" +
						"👉 Bạn có thể bấm nút **Nhập trực tiếp mã ngành** cho nhanh, hoặc tick chọn từng bước bên dưới:\n\n" +
						"**BƯỚC 1/2: CHỌN CÁC NGÀNH NGHỀ BỔ SUNG**",
					Components: []discordgo.MessageComponent{
						discordgo.ActionsRow{
							Components: []discordgo.MessageComponent{
								discordgo.Button{
									Label:    "⌨️ Nhập trực tiếp mã ngành Bổ sung & Bỏ",
									Style:    discordgo.PrimaryButton,
									CustomID: "btn_open_modal_both:" + capDoiFlag,
									Emoji:    &discordgo.ComponentEmoji{Name: "⚡"},
								},
							},
						},
						discordgo.ActionsRow{
							Components: []discordgo.MessageComponent{
								discordgo.SelectMenu{
									CustomID:    "select_nganh_step1_bs:" + capDoiFlag,
									Placeholder: "Hoặc tick chọn các ngành BỔ SUNG từ Drop list...",
									MinValues:   &minV,
									MaxValues:   maxV,
									Options:     opts,
								},
							},
						},
					},
				},
			})
			if err != nil {
				fmt.Printf("Lỗi gửi select menu bổ sung step 1: %v\n", err)
			}
			return
		}

		// Nhánh 2: Chỉ BỔ SUNG ngành nghề
		if hasBoSung {
			err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
				Type: discordgo.InteractionResponseChannelMessageWithSource,
				Data: &discordgo.InteractionResponseData{
					Content: "➕ **BỔ SUNG NGÀNH NGHỀ KINH DOANH**\n" +
						"👉 Bạn có thể bấm nút **Nhập trực tiếp mã ngành** cho nhanh, hoặc tick chọn từ danh sách:\n" +
						"*(Chọn xong bảng điền thông tin sẽ tự động mở)*",
					Components: []discordgo.MessageComponent{
						discordgo.ActionsRow{
							Components: []discordgo.MessageComponent{
								discordgo.Button{
									Label:    "⌨️ Nhập trực tiếp mã ngành Bổ sung (Nhanh)",
									Style:    discordgo.PrimaryButton,
									CustomID: "btn_open_modal_bs:" + capDoiFlag,
									Emoji:    &discordgo.ComponentEmoji{Name: "⚡"},
								},
							},
						},
						discordgo.ActionsRow{
							Components: []discordgo.MessageComponent{
								discordgo.SelectMenu{
									CustomID:    "select_nganh_bs_only:" + capDoiFlag,
									Placeholder: "Hoặc tick chọn các ngành BỔ SUNG từ Drop list...",
									MinValues:   &minV,
									MaxValues:   maxV,
									Options:     opts,
								},
							},
						},
					},
				},
			})
			if err != nil {
				fmt.Printf("Lỗi gửi select menu bổ sung only: %v\n", err)
			}
			return
		}

		// Nhánh 3: Chỉ BỎ BỚT ngành nghề
		if hasBo {
			err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
				Type: discordgo.InteractionResponseChannelMessageWithSource,
				Data: &discordgo.InteractionResponseData{
					Content: "➖ **BỎ BỚT NGÀNH NGHỀ KINH DOANH**\n" +
						"👉 Bạn có thể bấm nút **Nhập trực tiếp mã ngành** cho nhanh, hoặc tick chọn từ danh sách:\n" +
						"*(Chọn xong bảng điền thông tin sẽ tự động mở)*",
					Components: []discordgo.MessageComponent{
						discordgo.ActionsRow{
							Components: []discordgo.MessageComponent{
								discordgo.Button{
									Label:    "⌨️ Nhập trực tiếp mã ngành Bỏ bớt (Nhanh)",
									Style:    discordgo.PrimaryButton,
									CustomID: "btn_open_modal_bo:" + capDoiFlag,
									Emoji:    &discordgo.ComponentEmoji{Name: "⚡"},
								},
							},
						},
						discordgo.ActionsRow{
							Components: []discordgo.MessageComponent{
								discordgo.SelectMenu{
									CustomID:    "select_nganh_bo_only:" + capDoiFlag,
									Placeholder: "Hoặc tick chọn các ngành BỎ BỚT từ Drop list...",
									MinValues:   &minV,
									MaxValues:   maxV,
									Options:     opts,
								},
							},
						},
					},
				},
			})
			if err != nil {
				fmt.Printf("Lỗi gửi select menu bỏ only: %v\n", err)
			}
			return
		}

		// Nhánh 4: Không chọn ngành nghề nào (ví dụ: chỉ chọn Đổi tên, Trụ sở, Vốn, Lao động, Cấp đổi)
		labelThayDoi := "Nội dung thay đổi"
		placeholderThayDoi := "Nhập thông tin thay đổi mới"
		if hasTen {
			labelThayDoi = "Tên HKD mới (in hoa)"
			placeholderThayDoi = "VD: BÁNH ƯỚT GIA TRUYỀN PHÚ CƯỜNG"
		} else if hasTruSo {
			labelThayDoi = "Địa chỉ trụ sở mới & SĐT (Số nhà, đường)"
			placeholderThayDoi = "VD: 123 Đường Số 1, KP2 - SĐT: 0901234567"
		} else if hasVon {
			labelThayDoi = "Vốn mới & Hình thức thay đổi"
			placeholderThayDoi = "VD: 100.000.000 VNĐ - Tăng vốn"
		} else if hasLaoDong {
			labelThayDoi = "Tổng số lao động"
			placeholderThayDoi = "VD: 3"
		}

		modalCustomID := "modal_thaydoi_submit:other:" + capDoiFlag
		openModalForThayDoi(s, i, modalCustomID, labelThayDoi, placeholderThayDoi, "")

	case discordgo.InteractionModalSubmit:
		data := i.ModalSubmitData()

		// Xử lý modal nhập thêm thông tin địa chỉ {thongtinthem}
		if strings.HasPrefix(data.CustomID, "modal_thongtinthem_submit:") {
			targetUserID := strings.TrimPrefix(data.CustomID, "modal_thongtinthem_submit:")
			valTTT := ""
			for _, comp := range data.Components {
				if row, ok := comp.(*discordgo.ActionsRow); ok {
					for _, c := range row.Components {
						if input, ok := c.(*discordgo.TextInput); ok && input.CustomID == "thong_tin_them" {
							valTTT = input.Value
						}
					}
				}
			}
			_ = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
				Type: discordgo.InteractionResponseChannelMessageWithSource,
				Data: &discordgo.InteractionResponseData{
					Content: "⏳ **Đã ghi nhận thông tin thêm, đang hoàn tất văn bản...**",
				},
			})
			pendingThayDoiMu.Lock()
			pendingData, exists := pendingThayDoiSessions[targetUserID]
			pendingData.ThongTinThem = strings.TrimSpace(valTTT)
			pendingThayDoiSessions[targetUserID] = pendingData
			pendingThayDoiMu.Unlock()

			if exists {
				checkAndProceedThayDoi(s, i.ChannelID, targetUserID, pendingData)
			}
			return
		}

		// Xử lý modal nhập thêm Số lao động (Ô số 6 khi cấp đổi)
		if strings.HasPrefix(data.CustomID, "modal_solaodong_submit:") {
			targetUserID := strings.TrimPrefix(data.CustomID, "modal_solaodong_submit:")
			valLD := ""
			for _, comp := range data.Components {
				if row, ok := comp.(*discordgo.ActionsRow); ok {
					for _, c := range row.Components {
						if input, ok := c.(*discordgo.TextInput); ok && input.CustomID == "so_lao_dong" {
							valLD = input.Value
						}
					}
				}
			}
			_ = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
				Type: discordgo.InteractionResponseChannelMessageWithSource,
				Data: &discordgo.InteractionResponseData{
					Content: "⏳ **Đã ghi nhận số lao động, đang tạo văn bản Word...**",
				},
			})
			pendingThayDoiMu.Lock()
			pendingData, exists := pendingThayDoiSessions[targetUserID]
			delete(pendingThayDoiSessions, targetUserID)
			pendingThayDoiMu.Unlock()

			if exists {
				pendingData.SoLaoDong = strings.TrimSpace(valLD)
				generateAndSendThayDoiDocx(s, i.ChannelID, pendingData)
			}
			return
		}

		if !strings.HasPrefix(data.CustomID, "modal_thaydoi_submit") {
			return
		}

		values := make(map[string]string)
		for _, comp := range data.Components {
			if row, ok := comp.(*discordgo.ActionsRow); ok {
				for _, c := range row.Components {
					if input, ok := c.(*discordgo.TextInput); ok {
						values[input.CustomID] = input.Value
					}
				}
			}
		}

		// Phản hồi tạm tương tác
		_ = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: "⏳ **Đang xử lý tạo hồ sơ thay đổi, vui lòng đợi trong giây lát...**",
			},
		})

		// Xử lý dữ liệu từ Modal
		userID := getInteractionUserID(i)
		processThayDoiModal(s, i.ChannelID, userID, values, data.CustomID)
	}
}

func openModalForThayDoi(s *discordgo.Session, i *discordgo.InteractionCreate, modalCustomID, label5, placeholder5, defaultValue5 string) {
	if len(label5) > 45 {
		label5 = label5[:42] + "..."
	}
	if len(placeholder5) > 100 {
		placeholder5 = placeholder5[:97] + "..."
	}

	components := []discordgo.MessageComponent{
		// Ô 1: Họ và tên chủ hộ kinh doanh (RIÊNG)
		discordgo.ActionsRow{
			Components: []discordgo.MessageComponent{
				discordgo.TextInput{
					CustomID:    "ho_va_ten",
					Label:       "Họ và tên chủ hộ kinh doanh",
					Style:       discordgo.TextInputShort,
					Placeholder: "VD: NGUYỄN VĂN A",
					Required:    true,
				},
			},
		},
		// Ô 2: Tên hộ kinh doanh hiện tại (RIÊNG)
		discordgo.ActionsRow{
			Components: []discordgo.MessageComponent{
				discordgo.TextInput{
					CustomID:    "ten_hkd",
					Label:       "Tên hộ kinh doanh hiện tại",
					Style:       discordgo.TextInputShort,
					Placeholder: "VD: BÁNH ƯỚT PHÚ CƯỜNG",
					Required:    true,
				},
			},
		},
		// Ô 3: Cơ quan tiếp nhận (RIÊNG)
		discordgo.ActionsRow{
			Components: []discordgo.MessageComponent{
				discordgo.TextInput{
					CustomID:    "co_quan",
					Label:       "Cơ quan tiếp nhận (UBND Phường/Xã)",
					Style:       discordgo.TextInputShort,
					Placeholder: "VD: Phường Bình Tân",
					Required:    true,
				},
			},
		},
		// Ô 4: Mã số hộ kinh doanh / MST (RIÊNG)
		discordgo.ActionsRow{
			Components: []discordgo.MessageComponent{
				discordgo.TextInput{
					CustomID:    "ma_hkd",
					Label:       "Mã số hộ kinh doanh / MST",
					Style:       discordgo.TextInputShort,
					Placeholder: "VD: 0319988776",
					Required:    true,
				},
			},
		},
		// Ô 5: Nội dung thay đổi hoặc Mã ngành (RIÊNG)
		discordgo.ActionsRow{
			Components: []discordgo.MessageComponent{
				discordgo.TextInput{
					CustomID:    "noi_dung",
					Label:       label5,
					Style:       discordgo.TextInputParagraph,
					Placeholder: placeholder5,
					Value:       defaultValue5,
					Required:    true,
				},
			},
		},
	}

	err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseModal,
		Data: &discordgo.InteractionResponseData{
			CustomID:   modalCustomID,
			Title:      "Điền thông tin thay đổi HKD",
			Components: components,
		},
	})
	if err != nil {
		fmt.Printf("Lỗi mở modal thay đổi HKD: %v\n", err)
	}
}

func formatBoSungDefaultValue(codes []string) string {
	var sb strings.Builder
	for i, c := range codes {
		c = strings.TrimSpace(c)
		if c == "" {
			continue
		}
		ten := "Ngành nghề mã " + c
		defaultChiTiet := ""
		for _, db := range NganhNgheDB {
			if strconv.Itoa(db.MaNganh) == c {
				ten = strings.Split(db.TenNganh, "\n")[0]
				if strings.Contains(db.TenNganh, "Chi tiết:") {
					sp := strings.Split(db.TenNganh, "Chi tiết:")
					if len(sp) > 1 {
						defaultChiTiet = strings.TrimSpace(strings.ReplaceAll(sp[1], ")", ""))
						defaultChiTiet = strings.ReplaceAll(defaultChiTiet, "\n", " ")
					}
				}
				break
			}
		}
		chinhStr := ""
		if i == 0 {
			chinhStr = "(chính)"
		}
		chiTietVal := defaultChiTiet
		if chiTietVal == "" {
			chiTietVal = ten
		}
		// Cấu trúc: STT | Mã ngành | (sau đó mới điền chi tiết) | Tick box xem có phải ngành chính hay không
		sb.WriteString(fmt.Sprintf("%d | %s | %s | %s\n", i+1, c, chiTietVal, chinhStr))
	}
	return strings.TrimRight(sb.String(), "\n")
}

func formatBoDefaultValue(codes []string) string {
	var sb strings.Builder
	for i, c := range codes {
		c = strings.TrimSpace(c)
		if c == "" {
			continue
		}
		ten := "Ngành nghề mã " + c
		for _, db := range NganhNgheDB {
			if strconv.Itoa(db.MaNganh) == c {
				ten = strings.Split(db.TenNganh, "\n")[0]
				break
			}
		}
		sb.WriteString(fmt.Sprintf("%d | %s | %s\n", i+1, c, ten))
	}
	return strings.TrimRight(sb.String(), "\n")
}

func formatBothNganhDefaultValue(bsCodes, boCodes []string) string {
	var sb strings.Builder
	sb.WriteString("=== BỔ SUNG (STT | Mã | Chi tiết | (chính)) ===\n")
	sb.WriteString(formatBoSungDefaultValue(bsCodes))
	sb.WriteString("\n=== BỎ BỚT (STT | Mã | Tên ngành) ===\n")
	sb.WriteString(formatBoDefaultValue(boCodes))
	return sb.String()
}

func parseThayDoiVon(raw string) *model.ThayDoiVon {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil
	}

	hinhThuc := "Tăng vốn"
	lower := strings.ToLower(raw)
	if strings.Contains(lower, "giảm vốn") || strings.Contains(lower, "giam von") {
		hinhThuc = "Giảm vốn"
	}

	thoiGian := ""
	reDate := regexp.MustCompile(`\b\d{1,2}/\d{1,2}/\d{4}\b`)
	if match := reDate.FindString(raw); match != "" {
		thoiGian = match
	}

	clean := raw
	// Bỏ phần ngày tháng kèm chữ từ ngày, ngày, kể từ ngày
	reDatePhrase := regexp.MustCompile(`(?i)\(?(?:kể\s+từ\s+ngày|từ\s+ngày|ngày)?\s*\d{1,2}/\d{1,2}/\d{4}\)?`)
	clean = reDatePhrase.ReplaceAllString(clean, "")

	// Bỏ phần hình thức tăng vốn / giảm vốn
	reHinhThuc := regexp.MustCompile(`(?i)\(?(?:tăng\s+vốn|giảm\s+vốn|tang\s+von|giam\s+von)\)?`)
	clean = reHinhThuc.ReplaceAllString(clean, "")

	clean = strings.Trim(clean, ", -:;()[]{}")
	clean = strings.TrimSpace(clean)
	vonMoi := raw
	if clean != "" {
		vonMoi = clean
	}

	return &model.ThayDoiVon{
		VonKinhDoanhMoi: vonMoi,
		ThoiGianThayDoi: thoiGian,
		HinhThucThayDoi: hinhThuc,
	}
}

func parseBothNganhNghe(raw string) ([]model.NganhNgheKinhDoanh, []model.NganhNgheKinhDoanh) {
	var boSungList []model.NganhNgheKinhDoanh
	var boList []model.NganhNgheKinhDoanh

	lines := strings.Split(raw, "\n")
	currentMode := "bosung"

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		lower := strings.ToLower(line)
		if strings.Contains(lower, "bỏ bớt") || strings.Contains(lower, "bỏ:") || strings.Contains(lower, "bo:") || strings.HasPrefix(lower, "-") {
			currentMode = "bo"
			if strings.Contains(line, "===") {
				continue
			}
			line = strings.TrimLeft(line, "-+ ")
			line = strings.TrimPrefix(line, "Bỏ:")
			line = strings.TrimPrefix(line, "bỏ:")
			line = strings.TrimPrefix(line, "Bo:")
			line = strings.TrimPrefix(line, "bo:")
			line = strings.TrimPrefix(line, "Bỏ bớt:")
			line = strings.TrimPrefix(line, "bỏ bớt:")
		} else if strings.Contains(lower, "bổ sung") || strings.Contains(lower, "bo sung:") || strings.HasPrefix(lower, "+") || strings.Contains(lower, "thêm:") {
			currentMode = "bosung"
			if strings.Contains(line, "===") {
				continue
			}
			line = strings.TrimLeft(line, "-+ ")
			line = strings.TrimPrefix(line, "Bổ sung:")
			line = strings.TrimPrefix(line, "bổ sung:")
			line = strings.TrimPrefix(line, "Bo sung:")
			line = strings.TrimPrefix(line, "bo sung:")
			line = strings.TrimPrefix(line, "Thêm:")
			line = strings.TrimPrefix(line, "thêm:")
		}

		if line == "" || strings.Contains(line, "===") {
			continue
		}

		if currentMode == "bo" {
			boList = append(boList, parseNganhNgheWithChiTiet(line)...)
		} else {
			boSungList = append(boSungList, parseNganhNgheWithChiTiet(line)...)
		}
	}
	return boSungList, boList
}

func parseChiTietNganh(codes []string, rawText string) []model.NganhNgheKinhDoanh {
	var result []model.NganhNgheKinhDoanh
	rawText = strings.TrimSpace(rawText)

	lines := strings.Split(rawText, "\n")
	detailMap := make(map[string]string)
	isMainMap := make(map[string]bool)

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		subParts := strings.Split(line, ",")
		for _, sp := range subParts {
			sp = strings.TrimSpace(sp)
			if sp == "" {
				continue
			}
			matchedCode := ""
			for _, c := range codes {
				if strings.HasPrefix(sp, c+":") || strings.HasPrefix(sp, c+" -") || strings.HasPrefix(sp, c+" ") {
					matchedCode = c
					break
				}
			}

			if matchedCode != "" {
				rest := strings.TrimPrefix(sp, matchedCode)
				rest = strings.TrimPrefix(rest, ":")
				rest = strings.TrimPrefix(rest, "-")
				rest = strings.TrimSpace(rest)

				isMain := false
				lower := strings.ToLower(rest)
				if strings.Contains(lower, "(chính)") || strings.Contains(lower, "(chinh)") {
					isMain = true
					rest = strings.ReplaceAll(rest, "(chính)", "")
					rest = strings.ReplaceAll(rest, "(Chính)", "")
					rest = strings.ReplaceAll(rest, "(chinh)", "")
					rest = strings.ReplaceAll(rest, "(Chinh)", "")
					rest = strings.TrimSpace(rest)
				}
				detailMap[matchedCode] = rest
				if isMain {
					isMainMap[matchedCode] = true
				}
			}
		}
	}

	for _, codeStr := range codes {
		codeInt, _ := strconv.Atoi(codeStr)
		tenNganh := "Ngành nghề mã " + codeStr
		defaultChiTiet := ""

		for _, dbItem := range NganhNgheDB {
			if strconv.Itoa(dbItem.MaNganh) == codeStr {
				tenNganh = strings.Split(dbItem.TenNganh, "\n")[0]
				if strings.Contains(dbItem.TenNganh, "Chi tiết:") {
					sp := strings.Split(dbItem.TenNganh, "Chi tiết:")
					if len(sp) > 1 {
						defaultChiTiet = strings.TrimSpace(strings.ReplaceAll(sp[1], ")", ""))
						defaultChiTiet = strings.ReplaceAll(defaultChiTiet, "\n", " ")
					}
				}
				break
			}
		}

		chiTiet := detailMap[codeStr]
		isMain := isMainMap[codeStr]

		if len(codes) == 1 && chiTiet == "" && rawText != "" {
			rest := rawText
			lower := strings.ToLower(rest)
			if strings.Contains(lower, "(chính)") || strings.Contains(lower, "(chinh)") {
				isMain = true
				rest = strings.ReplaceAll(rest, "(chính)", "")
				rest = strings.ReplaceAll(rest, "(Chính)", "")
				rest = strings.ReplaceAll(rest, "(chinh)", "")
				rest = strings.ReplaceAll(rest, "(Chinh)", "")
				rest = strings.TrimSpace(rest)
			}
			chiTiet = rest
		}

		if chiTiet == "" {
			chiTiet = defaultChiTiet
		}

		result = append(result, model.NganhNgheKinhDoanh{
			TenNganh:     tenNganh,
			MaNganh:      codeInt,
			ChiTiet:      chiTiet,
			LaNganhChinh: isMain,
		})
	}

	return result
}

func parseBoNganh(codes []string) []model.NganhNgheKinhDoanh {
	var result []model.NganhNgheKinhDoanh
	for _, codeStr := range codes {
		codeInt, _ := strconv.Atoi(codeStr)
		tenNganh := "Ngành nghề mã " + codeStr
		for _, dbItem := range NganhNgheDB {
			if strconv.Itoa(dbItem.MaNganh) == codeStr {
				tenNganh = strings.Split(dbItem.TenNganh, "\n")[0]
				break
			}
		}
		result = append(result, model.NganhNgheKinhDoanh{
			TenNganh: tenNganh,
			MaNganh:  codeInt,
		})
	}
	return result
}

func processThayDoiModal(s *discordgo.Session, channelID, userID string, values map[string]string, customID string) {
	sendErr := func(msg string) { s.ChannelMessageSend(channelID, msg) }

	hoVaTen := strings.TrimSpace(values["ho_va_ten"])
	tenHKD := strings.TrimSpace(values["ten_hkd"])
	coQuan := strings.TrimSpace(values["co_quan"])
	maHKD := strings.TrimSpace(values["ma_hkd"])
	noiDung := strings.TrimSpace(values["noi_dung"])
	if noiDung == "" {
		noiDung = strings.TrimSpace(values["chi_tiet_nganh"])
	}
	if noiDung == "" {
		noiDung = strings.TrimSpace(values["noi_dung_thay_doi"])
	}

	// Fallback nếu người dùng dùng form cũ
	if tenHKD == "" || maHKD == "" {
		if infoChung := values["info_chung"]; infoChung != "" {
			parts := strings.Split(infoChung, ",")
			if len(parts) >= 1 && tenHKD == "" {
				tenHKD = strings.TrimSpace(parts[0])
			}
			if len(parts) >= 2 && maHKD == "" {
				maHKD = strings.TrimSpace(parts[1])
			}
			if len(parts) >= 3 && coQuan == "" {
				coQuan = strings.TrimSpace(parts[2])
			}
		}
	}

	if tenHKD == "" || maHKD == "" {
		sendErr("❌ Thiếu thông tin: Cần nhập tối thiểu `Tên HKD hiện tại` và `Mã HKD`!")
		return
	}

	hasCapDoi := strings.HasSuffix(customID, ":1") || strings.Contains(customID, ":capdoi")

	data := model.ThayDoiHKD{
		HoVaTen:   hoVaTen,
		TenHKD:    tenHKD,
		MaHKD:     maHKD,
		CoQuan:    coQuan,
		SoLaoDong: values["lao_dong"],
		CapDoiGCN: hasCapDoi,
	}

	// 1. Phân tích đổi tên
	if tenMoi := values["ten_moi"]; strings.TrimSpace(tenMoi) != "" {
		data.ThayDoiTen = &model.ThayDoiTenHKD{TenHKDMoi: strings.TrimSpace(tenMoi)}
	} else if strings.HasPrefix(strings.ToUpper(noiDung), "TÊN MỚI:") || strings.HasPrefix(strings.ToUpper(noiDung), "ĐỔI TÊN:") {
		tMoi := strings.TrimSpace(strings.TrimPrefix(strings.TrimPrefix(strings.ToUpper(noiDung), "TÊN MỚI:"), "ĐỔI TÊN:"))
		data.ThayDoiTen = &model.ThayDoiTenHKD{TenHKDMoi: tMoi}
	} else if strings.Contains(customID, ":other:") && !strings.Contains(noiDung, "Đường") && !strings.Contains(noiDung, "VNĐ") && !strings.Contains(noiDung, "đồng") && !strings.Contains(noiDung, "Vốn") && !strings.Contains(noiDung, "vốn") {
		data.ThayDoiTen = &model.ThayDoiTenHKD{TenHKDMoi: strings.TrimSpace(noiDung)}
	}

	// 2. Phân tích đổi trụ sở
	soNha := strings.TrimSpace(values["so_nha"])
	if soNha == "" && (strings.Contains(noiDung, "Trụ sở:") || strings.Contains(noiDung, "trụ sở:") || strings.Contains(noiDung, "Đường") || strings.Contains(noiDung, "đường") || strings.Contains(noiDung, "SĐT") || strings.Contains(noiDung, "Số nhà")) {
		soNha = noiDung
		if strings.Contains(noiDung, "Trụ sở:") {
			sp := strings.Split(noiDung, "Trụ sở:")
			if len(sp) > 1 {
				soNha = strings.TrimSpace(sp[1])
			}
		} else if strings.Contains(noiDung, "trụ sở:") {
			sp := strings.Split(noiDung, "trụ sở:")
			if len(sp) > 1 {
				soNha = strings.TrimSpace(sp[1])
			}
		}
	} else if truSoRaw := values["tru_so"]; strings.TrimSpace(truSoRaw) != "" && soNha == "" {
		soNha = truSoRaw
	}

	if soNha != "" {
		sdt := ""
		if strings.Contains(soNha, "SĐT:") || strings.Contains(soNha, "SĐT") || strings.Contains(soNha, "sdt:") {
			sp := strings.Split(soNha, "-")
			if len(sp) >= 2 {
				soNha = strings.TrimSpace(sp[0])
				sdt = strings.TrimSpace(strings.ReplaceAll(strings.ReplaceAll(sp[1], "SĐT:", ""), "SĐT", ""))
			}
		}
		data.ThayDoiTruSo = &model.ThayDoiTruSo{
			SoNha:    soNha,
			CoQuan:   coQuan,
			ThanhPho: "Thành phố Hồ Chí Minh",
			SDT:      sdt,
		}
	}

	// Kiểm tra nếu có nhập sẵn Thông tin thêm trong nội dung
	if strings.Contains(noiDung, "Thông tin thêm:") {
		sp := strings.Split(noiDung, "Thông tin thêm:")
		if len(sp) > 1 {
			data.ThongTinThem = strings.TrimSpace(sp[1])
		}
	}

	// 3. Phân tích đổi vốn
	if vonRaw := values["von"]; strings.TrimSpace(vonRaw) != "" {
		data.ThayDoiVon = parseThayDoiVon(vonRaw)
	} else if strings.Contains(noiDung, "VNĐ") || strings.Contains(noiDung, "đồng") || strings.Contains(strings.ToLower(noiDung), "vốn") {
		data.ThayDoiVon = parseThayDoiVon(noiDung)
	}

	// 4. Ngành nghề từ Drop list hoặc nhập mã trực tiếp
	var bsCodes []string
	var boCodes []string

	if strings.Contains(customID, ":bs=") {
		parts := strings.Split(customID, ":bs=")
		if len(parts) > 1 {
			sub := strings.Split(parts[1], ":")
			rawBs := sub[0]
			if rawBs != "" {
				for _, c := range strings.Split(rawBs, ",") {
					c = strings.TrimSpace(c)
					if c != "" {
						bsCodes = append(bsCodes, c)
					}
				}
			}
		}
	}

	if strings.Contains(customID, ":bo=") {
		parts := strings.Split(customID, ":bo=")
		if len(parts) > 1 {
			sub := strings.Split(parts[1], ":")
			rawBo := sub[0]
			if rawBo != "" {
				for _, c := range strings.Split(rawBo, ",") {
					c = strings.TrimSpace(c)
					if c != "" {
						boCodes = append(boCodes, c)
					}
				}
			}
		}
	}

	var boSungList []model.NganhNgheKinhDoanh
	var boList []model.NganhNgheKinhDoanh

	if len(bsCodes) > 0 && len(boCodes) > 0 {
		bsList, bList := parseBothNganhNghe(noiDung)
		boSungList = bsList
		boList = bList
		if len(boSungList) == 0 {
			boSungList = parseChiTietNganh(bsCodes, noiDung)
		}
		if len(boList) == 0 {
			boList = parseBoNganh(boCodes)
		}
	} else if len(bsCodes) > 0 {
		boSungList = parseNganhNgheWithChiTiet(noiDung)
		if len(boSungList) == 0 {
			boSungList = parseChiTietNganh(bsCodes, noiDung)
		}
	} else if len(boCodes) > 0 {
		boList = parseNganhNgheWithChiTiet(noiDung)
		if len(boList) == 0 {
			boList = parseBoNganh(boCodes)
		}
	}

	// Nhập trực tiếp mã ngành từ ô noi_dung (khi không dùng Drop list hoặc dùng nút Nhập nhanh)
	if len(bsCodes) == 0 && len(boCodes) == 0 && noiDung != "" {
		if strings.Contains(noiDung, "Bỏ:") || strings.Contains(noiDung, "bỏ:") || strings.Contains(noiDung, "Bo:") || strings.Contains(noiDung, "bo:") || strings.Contains(noiDung, "-") || strings.Contains(noiDung, "Bổ sung:") || strings.Contains(noiDung, "bổ sung:") || strings.Contains(noiDung, "===") {
			bsParsed, boParsed := parseBothNganhNghe(noiDung)
			boSungList = append(boSungList, bsParsed...)
			boList = append(boList, boParsed...)
		} else if strings.Contains(customID, ":bo=") {
			boList = append(boList, parseNganhNgheWithChiTiet(noiDung)...)
		} else if data.ThayDoiTruSo == nil && data.ThayDoiVon == nil && data.ThayDoiTen == nil {
			boSungList = append(boSungList, parseNganhNgheWithChiTiet(noiDung)...)
		}
	}

	if len(boSungList) > 0 || len(boList) > 0 {
		data.ThayDoiNganhNghe = &model.ThayDoiNganhNghe{
			BoSung: boSungList,
			Bo:     boList,
		}
	} else if nnRaw := values["nganh_nghe"]; strings.TrimSpace(nnRaw) != "" {
		data.ThayDoiNganhNghe = &model.ThayDoiNganhNghe{
			BoSung: parseNganhNgheWithChiTiet(nnRaw),
		}
	}

	// Bước hỏi thông tin thêm nếu có thay đổi địa chỉ trụ sở mới
	if data.ThayDoiTruSo != nil && strings.TrimSpace(data.ThongTinThem) == "" {
		pendingThayDoiMu.Lock()
		pendingThayDoiSessions[userID] = data
		pendingThayDoiMu.Unlock()

		msg := &discordgo.MessageSend{
			Content: fmt.Sprintf("🏠 **Hệ thống ghi nhận bạn đã thay đổi địa chỉ trụ sở mới:** `%s`\n\n"+
				"❓ **Bạn có muốn nhập thêm thông tin chi tiết (`{thongtinthem}`) cho địa chỉ này không?**\n"+
				"*(Ví dụ: số phòng, số tầng, tòa nhà, thông tin mô tả thêm...)*", data.ThayDoiTruSo.SoNha),
			Components: []discordgo.MessageComponent{
				discordgo.ActionsRow{
					Components: []discordgo.MessageComponent{
						discordgo.Button{
							Label:    "📝 Có, nhập thêm thông tin",
							Style:    discordgo.PrimaryButton,
							CustomID: "btn_prompt_thongtinthem:" + userID,
							Emoji:    &discordgo.ComponentEmoji{Name: "➕"},
						},
						discordgo.Button{
							Label:    "⚡ Không, tiếp tục ngay",
							Style:    discordgo.SecondaryButton,
							CustomID: "btn_skip_thongtinthem:" + userID,
							Emoji:    &discordgo.ComponentEmoji{Name: "⏭️"},
						},
					},
				},
			},
		}
		_, err := s.ChannelMessageSendComplex(channelID, msg)
		if err != nil {
			// Fallback nếu không gửi được menu nút bấm
			checkAndProceedThayDoi(s, channelID, userID, data)
		}
		return
	}

	checkAndProceedThayDoi(s, channelID, userID, data)
}

func checkAndProceedThayDoi(s *discordgo.Session, channelID, userID string, data model.ThayDoiHKD) {
	// Kiểm tra nếu có cấp đổi GCN và chưa có số lao động -> hỏi Ô số 6
	if data.CapDoiGCN && strings.TrimSpace(data.SoLaoDong) == "" {
		pendingThayDoiMu.Lock()
		pendingThayDoiSessions[userID] = data
		pendingThayDoiMu.Unlock()

		msg := &discordgo.MessageSend{
			Content: "📑 **ĐĂNG KÝ CẤP ĐỔI SANG GIẤY CHỨNG NHẬN ĐĂNG KÝ HỘ KINH DOANH**\n\n" +
				"❓ **Bạn có muốn điền thêm Số lao động (Ô số 6 - Mục 5.2) không?**\n" +
				"*(Nếu không điền, mục này sẽ để trống hoàn toàn)*",
			Components: []discordgo.MessageComponent{
				discordgo.ActionsRow{
					Components: []discordgo.MessageComponent{
						discordgo.Button{
							Label:    "👥 Điền Số lao động (Ô số 6)",
							Style:    discordgo.PrimaryButton,
							CustomID: "btn_prompt_solaodong:" + userID,
							Emoji:    &discordgo.ComponentEmoji{Name: "✏️"},
						},
						discordgo.Button{
							Label:    "⚡ Để trống & Xuất file Word ngay",
							Style:    discordgo.SuccessButton,
							CustomID: "btn_skip_solaodong:" + userID,
							Emoji:    &discordgo.ComponentEmoji{Name: "📄"},
						},
					},
				},
			},
		}
		_, err := s.ChannelMessageSendComplex(channelID, msg)
		if err != nil {
			pendingThayDoiMu.Lock()
			delete(pendingThayDoiSessions, userID)
			pendingThayDoiMu.Unlock()
			generateAndSendThayDoiDocx(s, channelID, data)
		}
		return
	}

	pendingThayDoiMu.Lock()
	delete(pendingThayDoiSessions, userID)
	pendingThayDoiMu.Unlock()

	generateAndSendThayDoiDocx(s, channelID, data)
}

func generateAndSendThayDoiDocx(s *discordgo.Session, channelID string, data model.ThayDoiHKD) {
	sendErr := func(msg string) { s.ChannelMessageSend(channelID, msg) }

	filePath, err := FillThayDoiTemplate(data, ThayDoiTemplatePath)
	if err != nil {
		sendErr("❌ Lỗi tạo văn bản thay đổi: " + err.Error())
		return
	}

	// Thống kê gửi lại
	var report strings.Builder
	report.WriteString(fmt.Sprintf("✅ **Đã tạo hồ sơ thay đổi thành công cho: %s**\n", strings.ToUpper(data.TenHKD)))
	if data.HoVaTen != "" {
		report.WriteString(fmt.Sprintf("👤 **Họ và tên chủ hộ:** %s\n", strings.ToUpper(data.HoVaTen)))
	}
	if data.ThayDoiTen != nil {
		report.WriteString(fmt.Sprintf("🏷️ **Tên mới:** %s\n", strings.ToUpper(data.ThayDoiTen.TenHKDMoi)))
	}
	if data.ThayDoiTruSo != nil {
		report.WriteString(fmt.Sprintf("🏠 **Trụ sở mới:** %s (SĐT: %s)\n", data.ThayDoiTruSo.SoNha, data.ThayDoiTruSo.SDT))
		if data.ThongTinThem != "" {
			report.WriteString(fmt.Sprintf("📝 **Thông tin thêm địa chỉ:** %s\n", data.ThongTinThem))
		}
	}
	if data.ThayDoiVon != nil {
		report.WriteString(fmt.Sprintf("💰 **Vốn mới:** %s (%s, %s)\n", data.ThayDoiVon.VonKinhDoanhMoi, data.ThayDoiVon.HinhThucThayDoi, data.ThayDoiVon.ThoiGianThayDoi))
	}
	if data.ThayDoiNganhNghe != nil {
		if len(data.ThayDoiNganhNghe.BoSung) > 0 {
			report.WriteString(fmt.Sprintf("\n➕ **Đã bổ sung %d ngành nghề:**\n", len(data.ThayDoiNganhNghe.BoSung)))
			for idx, nn := range data.ThayDoiNganhNghe.BoSung {
				chinhTag := ""
				if nn.LaNganhChinh {
					chinhTag = " ⭐ *(Ngành chính)*"
				}
				report.WriteString(fmt.Sprintf("%d. [%d] %s%s\n", idx+1, nn.MaNganh, formatTenNganhWithChiTiet(nn.TenNganh, nn.ChiTiet), chinhTag))
			}
		}
		if len(data.ThayDoiNganhNghe.Bo) > 0 {
			report.WriteString(fmt.Sprintf("\n➖ **Đã bỏ bớt %d ngành nghề:**\n", len(data.ThayDoiNganhNghe.Bo)))
			for idx, nn := range data.ThayDoiNganhNghe.Bo {
				report.WriteString(fmt.Sprintf("%d. [%d] %s\n", idx+1, nn.MaNganh, nn.TenNganh))
			}
		}
	}
	if data.SoLaoDong != "" {
		report.WriteString(fmt.Sprintf("👥 **Số lao động:** %s\n", data.SoLaoDong))
	}
	if data.CapDoiGCN {
		report.WriteString("📑 **Cấp đổi sang GCN HKD:** Đã đánh dấu tick [X] vào ô Cá nhân và điền thông tin\n")
	}

	// Lưu file trực tiếp vào thư mục đích giống như lúc đăng ký thành lập (không gửi lên tránh tốn công tải về)
	s.ChannelMessageSend(channelID, report.String()+"\n📁 **Hồ sơ đã được lưu trực tiếp tại thư mục đích:**\n`"+filePath+"`")
}

func handleSubmitThayDoi(s *discordgo.Session, m *discordgo.MessageCreate) {
	sendErr := func(msg string) { s.ChannelMessageSend(m.ChannelID, msg) }

	raw := strings.TrimPrefix(m.Content, "!submit_thaydoi")
	fields := parseFields(raw)

	tenHKD := get(fields, "Tên HKD hiện tại", get(fields, "Tên HKD"))
	maHKD := get(fields, "Mã HKD", get(fields, "Mã số thuế"))
	coQuan := get(fields, "Cơ quan", get(fields, "Phường mới"))

	if tenHKD == "" || maHKD == "" {
		sendErr("❌ Vui lòng nhập tối thiểu: `Tên HKD hiện tại` và `Mã HKD`!")
		return
	}

	capDoi := strings.EqualFold(get(fields, "Cấp đổi"), "Có") || strings.EqualFold(get(fields, "Đăng ký cấp đổi"), "Có") || get(fields, "Cấp đổi") == "1"

	hoVaTen := get(fields, "Họ và tên", get(fields, "Họ tên", get(fields, "Chủ hộ")))

	data := model.ThayDoiHKD{
		HoVaTen:   hoVaTen,
		TenHKD:    tenHKD,
		MaHKD:     maHKD,
		CoQuan:    coQuan,
		SoLaoDong: get(fields, "Số lao động"),
		CapDoiGCN: capDoi,
	}

	// Đổi tên
	if tenMoi := get(fields, "Tên HKD mới"); tenMoi != "" {
		data.ThayDoiTen = &model.ThayDoiTenHKD{TenHKDMoi: tenMoi}
	}

	// Đổi trụ sở
	soNha := get(fields, "Số nhà mới")
	phuongMoi := get(fields, "Phường mới")
	thanhPho := get(fields, "Thành phố mới", "Thành phố Hồ Chí Minh")
	sdt := get(fields, "Số điện thoại")
	if soNha != "" || phuongMoi != "" || sdt != "" {
		data.ThayDoiTruSo = &model.ThayDoiTruSo{
			SoNha:    soNha,
			CoQuan:   phuongMoi,
			ThanhPho: thanhPho,
			SDT:      sdt,
		}
	}

	// Đổi vốn
	vonMoi := get(fields, "Vốn kinh doanh mới")
	tgVon := get(fields, "Thời gian thay đổi vốn")
	htVon := get(fields, "Hình thức thay đổi vốn")
	if vonMoi != "" || tgVon != "" || htVon != "" {
		data.ThayDoiVon = &model.ThayDoiVon{
			VonKinhDoanhMoi: vonMoi,
			ThoiGianThayDoi: tgVon,
			HinhThucThayDoi: htVon,
		}
	}

	// Ngành nghề
	boSungRaw := get(fields, "Ngành nghề bổ sung")
	boRaw := get(fields, "Ngành nghề bỏ")
	suaRaw := get(fields, "Ngành nghề sửa")

	boSungList := parseNganhNgheWithChiTiet(boSungRaw)
	boList := parseNganhNgheWithChiTiet(boRaw)
	suaList := parseNganhNgheWithChiTiet(suaRaw)

	if len(boSungList) > 0 || len(boList) > 0 || len(suaList) > 0 {
		data.ThayDoiNganhNghe = &model.ThayDoiNganhNghe{
			BoSung: boSungList,
			Bo:     boList,
			Sua:    suaList,
		}
	}

	data.ThongTinThem = get(fields, "Thông tin thêm", get(fields, "Thông tin thêm địa chỉ"))

	// Tạo file DOCX & gửi kết quả
	generateAndSendThayDoiDocx(s, m.ChannelID, data)
}

func parseNganhNgheWithChiTiet(raw string) []model.NganhNgheKinhDoanh {
	if strings.TrimSpace(raw) == "" {
		return nil
	}
	var results []model.NganhNgheKinhDoanh

	lines := strings.Split(raw, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "===") {
			continue
		}

		// Nếu dòng chứa format cột "|" (STT | Mã ngành | Chi tiết | (chính))
		if strings.Contains(line, "|") {
			parts := strings.Split(line, "|")
			var rawParts []string
			for _, p := range parts {
				p = strings.TrimSpace(p)
				if p != "" {
					rawParts = append(rawParts, p)
				}
			}

			isMain := false
			for _, p := range rawParts {
				lowerP := strings.ToLower(p)
				if lowerP == "(chính)" || lowerP == "chính" || lowerP == "(chinh)" || lowerP == "chinh" || lowerP == "x" || lowerP == "v" || strings.Contains(lowerP, "chính") || strings.Contains(lowerP, "chinh") {
					isMain = true
				}
			}

			var maStr string
			// Tìm mã ngành trong các part
			for _, p := range rawParts {
				cleanP := strings.TrimPrefix(p, "Mã:")
				cleanP = strings.TrimPrefix(cleanP, "mã:")
				cleanP = strings.TrimPrefix(cleanP, "Mã ngành:")
				cleanP = strings.TrimPrefix(cleanP, "mã ngành:")
				cleanP = strings.TrimSpace(cleanP)
				if num, err := strconv.Atoi(cleanP); err == nil && num >= 100 {
					maStr = cleanP
					break
				}
			}
			if maStr == "" && len(rawParts) >= 2 {
				maStr = rawParts[1]
			} else if maStr == "" && len(rawParts) == 1 {
				maStr = rawParts[0]
			}

			var detailParts []string
			for _, p := range rawParts {
				lowerP := strings.ToLower(p)
				if lowerP == "(chính)" || lowerP == "chính" || lowerP == "(chinh)" || lowerP == "chinh" || lowerP == "x" || lowerP == "v" {
					continue
				}
				cleanP := strings.TrimPrefix(p, "Mã:")
				cleanP = strings.TrimPrefix(cleanP, "mã:")
				cleanP = strings.TrimPrefix(cleanP, "Mã ngành:")
				cleanP = strings.TrimPrefix(cleanP, "mã ngành:")
				cleanP = strings.TrimSpace(cleanP)
				if cleanP == maStr {
					continue
				}
				if num, err := strconv.Atoi(cleanP); err == nil && num < 100 {
					continue // Số thứ tự STT
				}
				if strings.HasPrefix(lowerP, "stt ") || lowerP == "stt" {
					continue
				}
				p = strings.TrimPrefix(p, "Chi tiết:")
				p = strings.TrimPrefix(p, "chi tiết:")
				p = strings.TrimSpace(p)
				if p != "" {
					detailParts = append(detailParts, p)
				}
			}

			chiTiet := strings.Join(detailParts, " - ")
			addNganhToResults(&results, maStr, chiTiet, isMain)
			continue
		}

		// Dòng không chứa "|": phân tách bằng dấu phẩy hoặc chấm phẩy
		tokens := strings.FieldsFunc(line, func(r rune) bool {
			return r == ',' || r == ';'
		})
		for _, token := range tokens {
			item := strings.TrimSpace(token)
			if item == "" {
				continue
			}

			isMain := false
			lower := strings.ToLower(item)
			if strings.Contains(lower, "(chính)") || strings.Contains(lower, "(chinh)") {
				isMain = true
				item = strings.ReplaceAll(item, "(chính)", "")
				item = strings.ReplaceAll(item, "(Chính)", "")
				item = strings.ReplaceAll(item, "(chinh)", "")
				item = strings.ReplaceAll(item, "(Chinh)", "")
				item = strings.TrimSpace(item)
			}

			var maStr, chiTiet string
			if strings.Contains(item, ":") {
				sp := strings.SplitN(item, ":", 2)
				maStr = strings.TrimSpace(sp[0])
				chiTiet = strings.TrimSpace(sp[1])
			} else if strings.Contains(item, "-") {
				sp := strings.SplitN(item, "-", 2)
				maStr = strings.TrimSpace(sp[0])
				chiTiet = strings.TrimSpace(sp[1])
			} else {
				maStr = strings.TrimSpace(item)
			}

			addNganhToResults(&results, maStr, chiTiet, isMain)
		}
	}

	return results
}

func addNganhToResults(results *[]model.NganhNgheKinhDoanh, maStr, chiTiet string, isMain bool) {
	if maStr == "" {
		return
	}
	matched := false
	for _, dbItem := range NganhNgheDB {
		if strconv.Itoa(dbItem.MaNganh) == maStr || strings.EqualFold(dbItem.TenNganh, maStr) {
			*results = append(*results, model.NganhNgheKinhDoanh{
				TenNganh:     dbItem.TenNganh,
				MaNganh:      dbItem.MaNganh,
				ChiTiet:      chiTiet,
				LaNganhChinh: isMain,
			})
			matched = true
			break
		}
	}
	if !matched && maStr != "" {
		maInt, _ := strconv.Atoi(maStr)
		ten := maStr
		if maInt > 0 {
			ten = "Ngành nghề mã " + maStr
		}
		*results = append(*results, model.NganhNgheKinhDoanh{
			TenNganh:     ten,
			MaNganh:      maInt,
			ChiTiet:      chiTiet,
			LaNganhChinh: isMain,
		})
	}
}

// ==================== PARSE UTILS ====================

func parseFields(raw string) map[string]string {
	res := make(map[string]string)
	for _, e := range strings.FieldsFunc(raw, func(r rune) bool { return r == ';' || r == '\n' || r == '\r' }) {
		if kv := strings.SplitN(strings.TrimSpace(e), "=", 2); len(kv) == 2 {
			res[kv[0]] = kv[1]
		}
	}
	return res
}

func get(m map[string]string, key string, defaultVal ...string) string {
	if val, ok := m[key]; ok && val != "" {
		return val
	}
	if len(defaultVal) > 0 {
		return defaultVal[0]
	}
	return ""
}

func parseAddress(full string) model.DiaChi {
	parts := strings.Split(full, ",")
	dc := model.DiaChi{}
	if len(parts) >= 1 {
		dc.SoNha = strings.TrimSpace(parts[0])
	}
	if len(parts) >= 2 {
		dc.XaPhuong = strings.TrimSpace(parts[1])
	}
	if len(parts) >= 3 {
		dc.TinhTP = strings.TrimSpace(parts[2])
	}
	return dc
}

func searchNganhNgheByMaNganh(maNganh string) []model.NganhNgheKinhDoanh {
	var result []model.NganhNgheKinhDoanh
	for _, id := range strings.Split(maNganh, ",") {
		tid := strings.TrimSpace(id)
		for _, nganh := range NganhNgheDB {
			if strings.EqualFold(nganh.TenNganh, tid) || strings.EqualFold(strconv.Itoa(nganh.MaNganh), tid) {
				result = append(result, nganh)
				break
			}
		}
	}
	return result
}
