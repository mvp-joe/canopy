const greet = (name: string): string => {
  return "Hello, " + name;
};

const add = (a: number, b: number): number => a + b;

const double = (n: number): number => n * 2;

function apply(fn: (x: number) => number, value: number): number {
  return fn(value);
}

function main(): void {
  const msg = greet("world");
  const sum = add(1, 2);
  const result = apply(double, 5);
}
