package main

import (
	"fmt"
	"hkd.nam2507/model"
	"hkd.nam2507/service"
)

func main() {
	fmt.Println("⏳ Đang test submit...")

	sample := model.Hokinhdoanh{
		HoVaTen:        "Nguyễn Văn A",
		TenHoKinhDoanh: "TEST HKD",
		SDT:            "0123456789",
		DiaChiKinhDoanh: model.DiaChi{
			SoNha:    "123 Đường ABC",
			XaPhuong: "Phường Bình Hưng Hòa",
			TinhTP:   "Hồ Chí Minh",
		},
		NganhNgheKinhDoanh: []model.NganhNgheKinhDoanh{
			{TenNganh: "May trang phục", MaNganh: 1410},
		},
	}

	if err := service.SubmitHoSo(sample); err != nil {
		fmt.Println("❌ Lỗi:", err)
		return
	}

	fmt.Println("✅ Thành công!")
}
