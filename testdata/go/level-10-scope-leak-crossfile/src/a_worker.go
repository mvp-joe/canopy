package demo

func StartWorker() {
	cfg := loadWorker()
	_ = cfg
}

func loadWorker() string { return "default" }
