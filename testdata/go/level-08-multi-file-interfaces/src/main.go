package animals

func Describe(a Animal) string {
	return a.Name() + " says " + a.Sound()
}
