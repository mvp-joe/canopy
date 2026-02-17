class Outer:
    class Inner:
        class DeepInner:
            def deep_method(self):
                return "deep"

        def inner_method(self):
            return "inner"

    def outer_method(self):
        return "outer"

def standalone():
    return "standalone"
