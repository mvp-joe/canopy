class Config:
    def __init__(self, host, port):
        self.host = host
        self.port = port


class Handler:
    def handle(self, req):
        raise NotImplementedError

    def close(self):
        raise NotImplementedError


class Server(Handler):
    def __init__(self, name):
        self.name = name

    def handle(self, req):
        return "ok"

    def close(self):
        pass


def new_server(name):
    return Server(name)
