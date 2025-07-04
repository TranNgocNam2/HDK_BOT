package service

import (
	"fmt"
	"github.com/bwmarrin/discordgo"
	"hkd.nam2507/model"
	"os"
	"strconv"
	"strings"
)

const HKDTemplatePath1 = "templates/1.docx"
const HKDTemplatePath2 = "templates/2.docx"

var NganhNgheDB = []model.NganhNgheKinhDoanh{
	{TenNganh: "May trang phục (trừ trang phục từ da lông thú)\n(Chi tiết: May mặc; Không tẩy, nhuộm, hồ, in trên các sản phẩm vải, sợi dệt, may, đan)\n", MaNganh: 1410},
	{TenNganh: "Giặt là, làm sạch các sản phẩm dệt và lông thú\n(Chi tiết: Giặt ủi)\n", MaNganh: 9620},
	{TenNganh: "Bán buôn thực phẩm \n(Chi tiết: Bán buôn thủy sản)\n", MaNganh: 4632},
	{TenNganh: "Sản xuất giường, tủ, bàn, ghế\n(Chi tiết: Gia công lắp ráp bàn, ghế gỗ)\n", MaNganh: 3100},
	{TenNganh: "Sản xuất mì ống, mì sợi và sản phẩm tương tự\n(Chi tiết: Sản xuất mì tươi)\n", MaNganh: 1074},
	{TenNganh: "Gia công cơ khí; xử lý và tráng phủ kim loại\n(Chi tiết: Gia công tiện, phay, bào; Không rèn, đúc, dập, cắt, gò, hàn, sơn, xi mạ điện, cán kéo kim loại)\n", MaNganh: 2592},
	{TenNganh: "In ấn (Chi tiết: In chuyển nhiệt)\n", MaNganh: 1811},
	{TenNganh: "Sản xuất món ăn, thức ăn chế biến sẵn\n(Chi tiết: Sản xuất đậu hủ)\n", MaNganh: 1075},
	{TenNganh: "Sản xuất nước đá\n(Chi tiết: Sản xuất nước đá viên)\n", MaNganh: 3530},
	{TenNganh: "Sản xuất trang phục dệt kim, đan móc \n(Chi tiết: Thêu vi tính)\n", MaNganh: 1430},
	{TenNganh: "Kinh doanh dịch vụ lưu trú khác (trừ lưu trú ngắn ngày)\n(Chi tiết: Nhà trọ cho công nhân)\n", MaNganh: 5590},
}

var (
	Token      = os.Getenv("DISCORD_BOT_TOKEN")
	outputPath = "output/"
)

func MessageCreate(s *discordgo.Session, m *discordgo.MessageCreate) {
	if m.Author.Bot {
		return
	}

	if strings.HasPrefix(m.Content, "!register_hkd1") {
		handleHKDRegistration1(s, m)
	} else if strings.HasPrefix(m.Content, "!help") {
		sendHelpMessage(s, m.ChannelID)
	} else if strings.HasPrefix(m.Content, "!ma") {
		listNganhNghe(s, m.ChannelID)
	}
	if strings.HasPrefix(m.Content, "!register_hkd2") {
		handleHKDRegistration2(s, m)
	}
}

func handleHKDRegistration1(s *discordgo.Session, m *discordgo.MessageCreate) {
	if m.Author.Bot {
		return
	}

	if strings.HasPrefix(m.Content, "!register_hkd") {
		parts := strings.TrimPrefix(m.Content, "!register_hkd")
		fields := parseFields(parts)
		// Required fields:
		fullName := get(fields, "Họ và tên")
		diaChiThuongTru := get(fields, "Địa chỉ thường trú")
		ngaySinh := get(fields, "Ngày sinh")
		ngayCap := get(fields, "Ngày cấp CCCD")

		if fullName == "" || diaChiThuongTru == "" || ngaySinh == "" || ngayCap == "" {
			s.ChannelMessageSend(m.ChannelID, "❌ Thiếu thông tin bắt buộc!")
			return
		}

		nganhNgheKinhDoanh := searchNganhNgheByMaNganh(get(fields, "Ngành nghề kinh doanh"))
		if len(nganhNgheKinhDoanh) == 0 {
			s.ChannelMessageSend(m.ChannelID, "❌ Ngành nghề kinh doanh không hợp lệ!")
			return
		}

		cccd := get(fields, "CCCD")
		if cccd == "" || len(strings.TrimSpace(cccd)) != 12 {
			s.ChannelMessageSend(m.ChannelID, "❌ CCCD phải có 12 chữ số!")
			return
		}
		mst := get(fields, "Mã số thuế", cccd)

		phone := get(fields, "Số điện thoại")
		if phone == "" || len(phone) != 10 {
			s.ChannelMessageSend(m.ChannelID, "❌ Số điện thoại phải có 10 chữ số!")
			return
		}

		// Optional fields autofilled with required defaults:
		diaChiLienLac := get(fields, "Địa chỉ liên lạc", diaChiThuongTru)
		diaChiKinhDoanh := get(fields, "Địa chỉ kinh doanh", diaChiLienLac)
		coQuan := get(fields, "Cơ quan", parseAddress(diaChiKinhDoanh).XaPhuong)
		sample := model.Hokinhdoanh{
			HoVaTen:        fullName,
			GioiTinh:       get(fields, "Giới tính", "Nam"),
			NgaySinh:       ngaySinh,
			CCCD:           cccd,
			CoQuan:         coQuan,
			CoQuanCap:      get(fields, "Nơi cấp CCCD", "Cục cảnh sát Quản lý hành chính về trật tự xã hội"),
			NgayCap:        ngayCap,
			DanToc:         get(fields, "Dân tộc", "Kinh"),
			MST:            mst,
			SDT:            phone,
			TenHoKinhDoanh: get(fields, "Tên hộ kinh doanh", fullName),
			VonKinhDoanh: model.VonKinhDoanh{
				BangSo:  get(fields, "Vốn kinh Doanh (Bằng Số)", "30.000.000"),
				BangChu: get(fields, "Vốn kinh Doanh (Bằng Chữ)", "Ba mươi"),
			},
			DiaChiThuongTru:    parseAddress(diaChiThuongTru),
			DiaChiLienLac:      parseAddress(diaChiLienLac),
			DiaChiKinhDoanh:    parseAddress(diaChiKinhDoanh),
			NganhNgheKinhDoanh: nganhNgheKinhDoanh,
		}

		output, err := fillHKDTemplate(sample, HKDTemplatePath1)
		if err != nil {
			fmt.Println("Error filling template:", err)
			s.ChannelMessageSend(m.ChannelID, "❌ Lỗi tạo tài liệu: "+err.Error())
			return
		}

		fmt.Println(phone)
		file, err := os.Open(output)
		if err != nil {
			fmt.Println("Error opening file:", err)
			s.ChannelMessageSend(m.ChannelID, "❌ Không thể đọc file tài liệu: "+err.Error())
			return
		}
		defer file.Close()

		s.ChannelFileSend(m.ChannelID, phone+".docx", file)
	}
}

func handleHKDRegistration2(s *discordgo.Session, m *discordgo.MessageCreate) {
	if m.Author.Bot {
		return
	}

	if strings.HasPrefix(m.Content, "!register_hkd") {
		parts := strings.TrimPrefix(m.Content, "!register_hkd")
		fields := parseFields(parts)
		// Required fields:
		fullName := get(fields, "Họ và tên")
		diaChiThuongTru := get(fields, "Địa chỉ thường trú")
		ngaySinh := get(fields, "Ngày sinh")
		ngayCap := get(fields, "Ngày cấp CCCD")

		if fullName == "" || diaChiThuongTru == "" || ngaySinh == "" || ngayCap == "" {
			s.ChannelMessageSend(m.ChannelID, "❌ Thiếu thông tin bắt buộc!")
			return
		}

		nganhNgheKinhDoanh := searchNganhNgheByMaNganh(get(fields, "Ngành nghề kinh doanh"))
		if len(nganhNgheKinhDoanh) == 0 {
			s.ChannelMessageSend(m.ChannelID, "❌ Ngành nghề kinh doanh không hợp lệ!")
			return
		}

		cccd := get(fields, "CCCD")
		if cccd == "" || len(strings.TrimSpace(cccd)) != 12 {
			s.ChannelMessageSend(m.ChannelID, "❌ CCCD phải có 12 chữ số!")
			return
		}

		phone := get(fields, "Số điện thoại")
		if phone == "" || len(phone) != 10 {
			s.ChannelMessageSend(m.ChannelID, "❌ Số điện thoại phải có 10 chữ số!")
			return
		}

		// Optional fields autofilled with required defaults:
		diaChiLienLac := get(fields, "Địa chỉ liên lạc", diaChiThuongTru)
		diaChiKinhDoanh := get(fields, "Địa chỉ kinh doanh", diaChiLienLac)
		coQuan := get(fields, "Cơ quan", parseAddress(diaChiKinhDoanh).XaPhuong)
		sample := model.Hokinhdoanh{
			HoVaTen:        fullName,
			GioiTinh:       get(fields, "Giới tính", "Nam"),
			NgaySinh:       ngaySinh,
			CCCD:           cccd,
			CoQuan:         coQuan,
			CoQuanCap:      get(fields, "Nơi cấp CCCD", "Cục cảnh sát Quản lý hành chính về trật tự xã hội"),
			NgayCap:        ngayCap,
			DanToc:         get(fields, "Dân tộc", "Kinh"),
			MST:            get(fields, "Mã số thuế"),
			SDT:            phone,
			TenHoKinhDoanh: get(fields, "Tên hộ kinh doanh", fullName),
			VonKinhDoanh: model.VonKinhDoanh{
				BangSo:  get(fields, "Vốn kinh Doanh (Bằng Số)", "30.000.000"),
				BangChu: get(fields, "Vốn kinh Doanh (Bằng Chữ)", "Ba mươi"),
			},
			DiaChiThuongTru:    parseAddress(diaChiThuongTru),
			DiaChiLienLac:      parseAddress(diaChiLienLac),
			DiaChiKinhDoanh:    parseAddress(diaChiKinhDoanh),
			NganhNgheKinhDoanh: nganhNgheKinhDoanh,
		}

		output, err := fillHKDTemplate(sample, HKDTemplatePath2)
		if err != nil {
			fmt.Println("Error filling template:", err)
			s.ChannelMessageSend(m.ChannelID, "❌ Lỗi tạo tài liệu: "+err.Error())
			return
		}

		fmt.Println(phone)
		file, err := os.Open(output)
		if err != nil {
			fmt.Println("Error opening file:", err)
			s.ChannelMessageSend(m.ChannelID, "❌ Không thể đọc file tài liệu: "+err.Error())
			return
		}
		defer file.Close()

		s.ChannelFileSend(m.ChannelID, phone+".docx", file)
	}
}

func sendHelpMessage(s *discordgo.Session, channelID string) {
	helpMsg := `**🏢 Hướng dẫn sử dụng Bot Đăng ký Hộ Kinh Doanh**

**Lệnh chính:**
` + "`!register_hkd1`" + ` - Tạo giấy đề nghị đăng ký hộ kinh doanh (sử dụng mẫu 1)

**Cú pháp:**
` + "```" + `
!register_hkd1
Họ và tên=..................
Giới tính=Nữ
Dân tộc: Kinh
Ngày sinh=../../....
Mã số thuế=............
Địa chỉ thường trú=.............................., ...................., .......................
Ngành nghề kinh doanh=....
CCCD=............
Ngày cấp CCCD=../../....
Số điện thoại=...............
Tên hộ kinh doanh=.........................
Địa chỉ kinh doanh=........................., Phường ...................., Thành phố Hồ Chí Minh
Địa chỉ liên lạc=................................, Phường ...................., Thành phố Hồ Chí Minh
Vốn kinh Doanh (Bằng Số)=20.000.000
Vốn kinh Doanh (Bằng Chữ)=Hai mươi
` + "```" + `

**Các trường bắt buộc:** Họ và tên, CCCD (12 số), Số điện thoại (10 số), Ngày sinh, Ngày cấp CCCD, Địa chỉ thường trú, Ngành nghề kinh doanh

**Lệnh chính:**
` + "`!register_hkd2`" + ` - Tạo giấy đề nghị đăng ký hộ kinh doanh (sử dụng mẫu 2 (Phường Bình Tân))

**Cú pháp:**
` + "```" + `
!register_hkd2
Họ và tên=..................
Giới tính=Nữ
Dân tộc: Kinh
Ngày sinh=../../....
Mã số thuế=............
Địa chỉ thường trú=.............................., ...................., .......................
Ngành nghề kinh doanh=....
CCCD=............
Ngày cấp CCCD=../../....
Số điện thoại=...............
Tên hộ kinh doanh=.........................
Địa chỉ kinh doanh=........................., Phường ...................., Thành phố Hồ Chí Minh
Địa chỉ liên lạc=................................, Phường ...................., Thành phố Hồ Chí Minh
Vốn kinh Doanh (Bằng Số)=20.000.000
Vốn kinh Doanh (Bằng Chữ)=Hai mươi
` + "```" + `

**Các trường bắt buộc:** Họ và tên, CCCD (12 số), Số điện thoại (10 số), Ngày sinh, Ngày cấp CCCD, Địa chỉ thường trú, Ngành nghề kinh doanh


**Lệnh khác:**
• ` + "`!ma`" + ` - Xem danh sách mã ngành nghề
• ` + "`!help`" + ` - Hiển thị hướng dẫn này`

	s.ChannelMessageSend(channelID, helpMsg)
}

func validateRequiredFields(fields map[string]string) []string {
	var errors []string

	requiredFields := map[string]string{
		"Họ và tên":             "",
		"Địa chỉ thường trú":    "",
		"Ngày sinh":             "",
		"Ngày cấp CCCD":         "",
		"CCCD":                  "12 chữ số",
		"Số điện thoại":         "10 chữ số",
		"Ngành nghề kinh doanh": "",
	}

	for field := range requiredFields {
		value := get(fields, field)
		if value == "" {
			errors = append(errors, fmt.Sprintf("• %s không được để trống", field))
			continue
		}

		// Specific validations
		switch field {
		case "CCCD":
			if len(strings.TrimSpace(value)) != 12 {
				errors = append(errors, fmt.Sprintf("• %s phải có đúng 12 chữ số", field))
			}
		case "Số điện thoại":
			if len(strings.TrimSpace(value)) != 10 {
				errors = append(errors, fmt.Sprintf("• %s phải có đúng 10 chữ số", field))
			}
		}
	}

	return errors
}

func buildHoKinhDoanhModel(fields map[string]string, nganhNgheKinhDoanh []model.NganhNgheKinhDoanh) model.Hokinhdoanh {
	fullName := get(fields, "Họ và tên")
	diaChiThuongTru := get(fields, "Địa chỉ thường trú")
	diaChiLienLac := get(fields, "Địa chỉ liên lạc", diaChiThuongTru)
	diaChiKinhDoanh := get(fields, "Địa chỉ kinh doanh", diaChiLienLac)

	return model.Hokinhdoanh{
		HoVaTen:        fullName,
		GioiTinh:       get(fields, "Giới tính", "Nam"),
		NgaySinh:       get(fields, "Ngày sinh"),
		CCCD:           get(fields, "CCCD"),
		CoQuan:         get(fields, "Cơ quan", "Phường Bình Tân"),
		CoQuanCap:      get(fields, "Nơi cấp CCCD", "Cục cảnh sát Quản lý hành chính về trật tự xã hội"),
		NgayCap:        get(fields, "Ngày cấp CCCD"),
		DanToc:         get(fields, "Dân tộc", "Kinh"),
		MST:            get(fields, "Mã số thuế"),
		SDT:            get(fields, "Số điện thoại"),
		TenHoKinhDoanh: get(fields, "Tên hộ kinh doanh", fullName),
		VonKinhDoanh: model.VonKinhDoanh{
			BangSo:  get(fields, "Vốn kinh doanh bằng số", "30.000.000"),
			BangChu: get(fields, "Vốn kinh doanh bằng chữ", "Ba mươi"),
		},
		DiaChiThuongTru:    parseAddress(diaChiThuongTru),
		DiaChiLienLac:      parseAddress(diaChiLienLac),
		DiaChiKinhDoanh:    parseAddress(diaChiKinhDoanh),
		NganhNgheKinhDoanh: nganhNgheKinhDoanh,
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

func parseFields(raw string) map[string]string {
	res := make(map[string]string)
	entries := strings.FieldsFunc(raw, func(r rune) bool {
		return r == ';' || r == '\n' || r == '\r'
	})
	for _, e := range entries {
		kv := strings.SplitN(strings.TrimSpace(e), "=", 2)
		if len(kv) == 2 {
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
	for i := range parts {
		parts[i] = strings.TrimSpace(parts[i])
	}
	dc := model.DiaChi{}
	if len(parts) >= 1 {
		dc.SoNha = parts[0]
	}
	if len(parts) >= 2 {
		dc.XaPhuong = parts[1]
	}
	if len(parts) >= 3 {
		dc.TinhTP = parts[2]
	}
	return dc
}

func searchNganhNgheByMaNganh(maNganh string) []model.NganhNgheKinhDoanh {
	var result []model.NganhNgheKinhDoanh
	ids := strings.Split(maNganh, ",")
	for _, id := range ids {
		tid := strings.TrimSpace(id) // Trim whitespace around the ID
		for _, nganh := range NganhNgheDB {
			if strings.EqualFold(nganh.TenNganh, tid) || strings.EqualFold(strconv.Itoa(nganh.MaNganh), tid) {
				result = append(result, nganh)
				break // Found a match, no need to check further
			}
		}

	}
	return result
}
