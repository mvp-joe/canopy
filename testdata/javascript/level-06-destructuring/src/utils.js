function formatUser({ name, age }) {
  return name + " (" + age + ")";
}

function sum([first, second, ...rest]) {
  let total = first + second;
  for (const n of rest) {
    total += n;
  }
  return total;
}

function createPair(a, b = 10) {
  return [a, b];
}

function logAll(...args) {
  for (const arg of args) {
    console.log(arg);
  }
}

function main() {
  formatUser({ name: "Alice", age: 30 });
  sum([1, 2, 3, 4]);
  createPair(5);
  logAll("a", "b", "c");
}
