export interface Readable {
  read(): string;
}

export interface Writable {
  write(data: string): void;
}

export interface ReadWrite extends Readable, Writable {
  flush(): void;
}
