package animals

type Dog struct {
	Breed string
}

func (d *Dog) Name() string {
	return "Dog"
}

func (d *Dog) Sound() string {
	return "Woof"
}

func (d *Dog) Move() string {
	return "run"
}

func NewDog(breed string) *Dog {
	return &Dog{Breed: breed}
}
