import { FileStream } from './stream';

function process(): void {
  const s = new FileStream();
  s.read();
  s.write("hello");
  s.flush();
}
