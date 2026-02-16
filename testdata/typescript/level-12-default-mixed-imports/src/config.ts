export default function createApp(name: string): void {
  console.log("App: " + name);
}

export function getVersion(): string {
  return "1.0";
}
