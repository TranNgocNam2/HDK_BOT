package model

type Hokinhdoanh struct {
	CoQuan             string               `json:"co_quan" binding:"required"`
	HoVaTen            string               `json:"ho_va_ten" binding:"required"`
	GioiTinh           string               `json:"gioi_tinh" binding:"required"`
	NgaySinh           string               `json:"ngay_sinh" binding:"required"`
	DanToc             string               `json:"dan_toc" binding:"required"`
	CCCD               string               `json:"cccd"`
	NgayCap            string               `json:"ngay_cap"`
	CoQuanCap          string               `json:"co_quan_cap"`
	DiaChiThuongTru    DiaChi               `json:"dia_chi_thuong_tru"`
	DiaChiLienLac      DiaChi               `json:"dia_chi_lien_lac"`
	SDT                string               `json:"sdt"`
	TenHoKinhDoanh     string               `json:"ten_ho_kinh_doanh"`
	DiaChiKinhDoanh    DiaChi               `json:"dia_chi_kinh_doanh"`
	NganhNgheKinhDoanh []NganhNgheKinhDoanh `json:"nganh_nghe_kinh_doanh"`
	VonKinhDoanh       VonKinhDoanh         `json:"von_kinh_doanh"`
}

type DiaChi struct {
	TinhTP   string `json:"tinh_tp"`
	XaPhuong string `json:"xa_phuong"`
	SoNha    string `json:"so_nha"`
}

type NganhNgheKinhDoanh struct {
	TenNganh     string `json:"ten_nganh"`
	MaNganh      int    `json:"ma_nganh"`
	ChiTiet      string `json:"chi_tiet,omitempty"`
	LaNganhChinh bool   `json:"la_nganh_chinh,omitempty"`
}

type VonKinhDoanh struct {
	BangSo  string `json:"bang_so"`
	BangChu string `json:"bang_chu"`
}

// ==================== THAY ĐỔI HKD ====================

type ThayDoiHKD struct {
	HoVaTen string `json:"ho_va_ten"`
	CoQuan  string `json:"co_quan"`
	TenHKD  string `json:"ten_hkd"`
	MaHKD   string `json:"ma_hkd"`

	ThayDoiTen       *ThayDoiTenHKD       `json:"thay_doi_ten,omitempty"`
	ThayDoiTruSo     *ThayDoiTruSo        `json:"thay_doi_tru_so,omitempty"`
	ThayDoiVon       *ThayDoiVon          `json:"thay_doi_von,omitempty"`
	ThayDoiNganhNghe *ThayDoiNganhNghe    `json:"thay_doi_nganh_nghe,omitempty"`
	SoLaoDong        string               `json:"so_lao_dong,omitempty"`
	CapDoiGCN        bool                 `json:"cap_doi_gcn,omitempty"`
	ThongTinThem     string               `json:"thong_tin_them,omitempty"`
}

type ThayDoiTenHKD struct {
	TenHKDMoi string `json:"ten_hkd_moi"`
}

type ThayDoiTruSo struct {
	SoNha    string `json:"sonha"`
	CoQuan   string `json:"co_quan"`
	ThanhPho string `json:"thanh_pho"`
	SDT      string `json:"sdt"`
}

type ThayDoiVon struct {
	VonKinhDoanhMoi   string `json:"vonkinhdoanhmoi"`
	ThoiGianThayDoi   string `json:"thoigianthaydoivon"`
	HinhThucThayDoi   string `json:"hinhthucthaydoivon"`
}

type ThayDoiNganhNghe struct {
	BoSung []NganhNgheKinhDoanh `json:"bo_sung,omitempty"`
	Bo     []NganhNgheKinhDoanh `json:"bo,omitempty"`
	Sua    []NganhNgheKinhDoanh `json:"sua,omitempty"`
}

// ==================== ỦY QUYỀN ====================

type UyQuyenData struct {
	HoVaTen         string `json:"ho_va_ten"`
	NgaySinh        string `json:"ngay_sinh"`
	CCCD            string `json:"cccd"`
	NgayCap         string `json:"ngay_cap"`
	CoQuanCap       string `json:"co_quan_cap"`
	DiaChiThuongTru string `json:"dia_chi_thuong_tru"`
	CoQuan          string `json:"co_quan"`
}

// ==================== WEB SUBMISSION ====================

type WebThayDoiSubmission struct {
	ThayDoi         ThayDoiHKD   `json:"thay_doi"`
	UyQuyen         *UyQuyenData `json:"uy_quyen,omitempty"`
	GCNImageBase64  string       `json:"gcn_image_base64,omitempty"`
	CCCDTruocBase64 string       `json:"cccd_truoc_base64,omitempty"`
	CCCDSauBase64   string       `json:"cccd_sau_base64,omitempty"`
}
