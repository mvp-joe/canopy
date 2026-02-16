import { Animal } from './animal';

export class Dog extends Animal {
  speak(): string {
    return "Woof!";
  }

  fetch(): string {
    return "Fetching...";
  }
}
