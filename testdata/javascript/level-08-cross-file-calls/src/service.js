import { info, warn } from './logger';

export function startService() {
  info("Service starting");
  return true;
}

export function stopService() {
  warn("Service stopping");
  return false;
}
