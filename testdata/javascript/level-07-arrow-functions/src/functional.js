const double = (x) => x * 2;

const greet = (name) => {
  return "Hello, " + name;
};

const identity = x => x;

function applyFn(value, fn) {
  return fn(value);
}

function compose(f, g) {
  return (x) => f(g(x));
}

function main() {
  const result1 = double(5);
  const result2 = greet("world");
  const result3 = applyFn(10, double);
  const composed = compose(double, double);
}
