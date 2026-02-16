class Outer:
    class Inner:
        def method(self):
            return "inner"

    def create_inner(self):
        return self.Inner()


class Tree:
    class Node:
        def __init__(self, value):
            self.value = value

        def display(self):
            return str(self.value)

    def __init__(self):
        self.root = None

    def add(self, value):
        self.root = self.Node(value)
