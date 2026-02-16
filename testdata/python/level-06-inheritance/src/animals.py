class Animal:
    def speak(self):
        return ""


class Flyable:
    def fly(self):
        return "flying"


class Bird(Animal, Flyable):
    def speak(self):
        return "chirp"


class Penguin(Bird):
    def speak(self):
        return "squawk"

    def fly(self):
        return "cannot fly"
