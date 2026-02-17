from dataclasses import dataclass
from typing import List, Optional
from calculator import Calculator
from formatter import format_result


@dataclass
class Result:
    value: float
    operation: str
    formatted: str


class MathService:
    def __init__(self):
        self.calculator = Calculator()
        self.history: List[Result] = []

    def compute(self, operation: str, a: float, b: float) -> Result:
        if operation == "add":
            value = self.calculator.add(a, b)
        elif operation == "subtract":
            value = self.calculator.subtract(a, b)
        elif operation == "multiply":
            value = self.calculator.multiply(a, b)
        elif operation == "divide":
            value = self.calculator.divide(a, b)
        else:
            raise ValueError(f"Unknown operation: {operation}")

        result = Result(
            value=value,
            operation=operation,
            formatted=format_result(value, operation),
        )
        self.history.append(result)
        return result

    def get_history(self) -> List[Result]:
        return list(self.history)

    def clear_history(self) -> None:
        self.history.clear()


def main():
    svc = MathService()
    r = svc.compute("add", 3, 4)
    print(f"{r.operation}: {r.formatted}")


if __name__ == "__main__":
    main()
