export function info(msg) {
  console.log("[INFO] " + msg);
}

export function warn(msg) {
  console.log("[WARN] " + msg);
}

export class Logger {
  constructor(prefix) {
    this.prefix = prefix;
  }

  log(msg) {
    console.log(this.prefix + ": " + msg);
  }
}
