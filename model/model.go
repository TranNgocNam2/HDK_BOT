package model

type Hokinhdoanh struct {
	CoQuan             string               `json:"co_quan" binding:"required"`
	HoVaTen            string               `json:"ho_va_ten" binding:"required"`
	GioiTinh           string               `json:"gioi_tinh" binding:"required"`
	NgaySinh           string               `json:"ngay_sinh" binding:"required"`
	DanToc             string               `json:"dan_toc" binding:"required"`
	MST                string               `json:"mst"`
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
	TinhTP    string `json:"tinh_tp"`
	QuanHuyen string `json:"quan_huyen"`
	XaPhuong  string `json:"xa_phuong"`
	SoNha     string `json:"so_nha"`
}

type NganhNgheKinhDoanh struct {
	//STT            string `json:"stt"`
	TenNganh       string `json:"ten_nganh_nghe"`
	MaNganh        int    `json:"ma_nganh"`
	NganhNgheChinh bool   `json:"nganh_nghe_chinh"`
}

type VonKinhDoanh struct {
	BangSo  string `json:"bang_so"`
	BangChu string `json:"bang_chu"`
}
