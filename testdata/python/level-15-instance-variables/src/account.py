class Account:
    def __init__(self, owner):
        self.owner = owner
        self.balance = 0

    def deposit(self, amount):
        self.balance = self.balance + amount

    def get_info(self):
        return self.owner + ": " + str(self.balance)


class SavingsAccount(Account):
    def __init__(self, owner, rate):
        self.rate = rate

    def apply_interest(self):
        self.deposit(self.balance * self.rate)
