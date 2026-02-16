import { startService, stopService } from './service';
import { Logger } from './logger';

function run() {
  startService();
  const log = new Logger("app");
  stopService();
}
