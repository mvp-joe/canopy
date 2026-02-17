def greet(name):
    return f"Hello, {name}!"

class Greeter:
    def __init__(self, prefix="Hello"):
        self.prefix = prefix

    def greet(self, name):
        return f"{self.prefix}, {name}!"
