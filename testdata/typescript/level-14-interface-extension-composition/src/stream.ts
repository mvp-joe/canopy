import { ReadWrite } from './interfaces';

export class FileStream implements ReadWrite {
  read(): string {
    return "data";
  }

  write(data: string): void {}

  flush(): void {}
}
