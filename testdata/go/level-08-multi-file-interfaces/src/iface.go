package animals

type Animal interface {
	Name() string
	Sound() string
}

type Mover interface {
	Move() string
}
