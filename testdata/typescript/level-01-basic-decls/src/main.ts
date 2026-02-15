const MaxRetries = 3;

let debug = false;

function hello(): string {
  return "hello";
}

function add(a: number, b: number): number {
  return a + b;
}

const result = add(1, 2);
