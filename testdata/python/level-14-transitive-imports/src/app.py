from middleware import process
from core import transform


def run(data):
    result = process(data)
    return transform(result)
