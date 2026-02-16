class Animal:
    def speak(self):
        return ""

    def greet(self):
        return self.speak()


class Dog(Animal):
    def speak(self):
        return "woof"


class Cat(Animal):
    def speak(self):
        return "meow"
