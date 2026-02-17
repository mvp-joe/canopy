export class Logger {
  private debug: boolean;

  constructor(debug: boolean) {
    this.debug = debug;
  }

  info(message: string): void {
    console.log(`[INFO] ${message}`);
  }

  error(message: string): void {
    console.error(`[ERROR] ${message}`);
  }

  warn(message: string): void {
    if (this.debug) {
      console.warn(`[WARN] ${message}`);
    }
  }
}
