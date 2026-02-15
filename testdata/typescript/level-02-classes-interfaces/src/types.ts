interface Handler {
  handle(req: string): string;
  close(): void;
}

class Server {
  name: string;

  constructor(name: string) {
    this.name = name;
  }

  handle(req: string): string {
    return "ok";
  }

  close(): void {}
}

function newServer(name: string): Server {
  return new Server(name);
}
