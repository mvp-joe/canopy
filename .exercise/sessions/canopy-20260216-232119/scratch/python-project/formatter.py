def format_result(value: float, operation: str) -> str:
    return f"{operation} = {value:.2f}"


def format_table(results: list) -> str:
    lines = []
    for r in results:
        lines.append(f"  {r.operation}: {r.value:.2f}")
    return "\n".join(lines)
