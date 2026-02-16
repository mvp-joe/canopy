from core import validate


def process(data):
    if validate(data):
        return data
    return ""
