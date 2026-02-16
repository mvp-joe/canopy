package typeflow

func TypeAssert(s Shape) string {
	c, ok := s.(Circle)
	if ok {
		return c.String()
	}
	return "not a circle"
}

func TypeSwitch(s Shape) float64 {
	switch v := s.(type) {
	case Circle:
		return v.Area()
	case Square:
		return v.Area()
	}
	return 0
}
