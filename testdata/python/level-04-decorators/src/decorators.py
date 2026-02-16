def log_calls(func):
    return func


class Cache:
    @staticmethod
    def clear():
        pass

    @classmethod
    def create(cls):
        return cls()

    @property
    def size(self):
        return 0

    @log_calls
    def get(self, key):
        return None
