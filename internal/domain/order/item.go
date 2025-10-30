package order

type Item struct {
	ChrtId      string `json:"chrt_id"`
	TrackNumber string `json:"track_number"`
	Price       int64  `json:"price"`
	Rid         string `json:"rid"`
	Name        string `json:"name"`
	Sale        int64  `json:"sale"`
	Size        int64  `json:"size"`
	TotalPrice  int64  `json:"total_price"`
	NmId        string `json:"nm_id"`
	Brand       string `json:"brand"`
	Status      int64  `json:"status"`
}
