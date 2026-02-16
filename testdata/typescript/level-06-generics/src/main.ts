interface Sortable {
  compareTo(other: Sortable): number;
}

class NumBox implements Sortable {
  value: number;

  constructor(v: number) {
    this.value = v;
  }

  compareTo(other: Sortable): number {
    return 0;
  }
}

function first<T>(items: T[]): T {
  return items[0];
}

function main(): void {
  const box = new NumBox(42);
  const result = first([1, 2, 3]);
}
