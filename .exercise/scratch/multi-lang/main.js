function greet(name) {
  return "Hello, " + name;
}

const MAX_RETRIES = 3;

let counter = 0;

var globalFlag = true;

const add = (a, b) => a + b;

const multiply = (a, b) => {
  return a * b;
};

function processItems(items, callback) {
  for (const item of items) {
    callback(item);
  }
}
