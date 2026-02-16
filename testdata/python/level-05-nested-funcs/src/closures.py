def outer():
    x = 10

    def inner():
        return x

    return inner()


def make_counter():
    count = 0

    def increment():
        nonlocal count
        count += 1
        return count

    def get_count():
        return count

    return increment, get_count
