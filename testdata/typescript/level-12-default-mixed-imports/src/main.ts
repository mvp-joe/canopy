import createApp, { getVersion } from './config';
import { info } from './logger';

function boot(): void {
  createApp("myApp");
  const ver = getVersion();
  info("started");
}
