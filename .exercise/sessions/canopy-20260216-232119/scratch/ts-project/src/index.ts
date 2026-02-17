import { UserRepository } from './repository';
import { Logger } from './logger';

interface Config {
  dbUrl: string;
  port: number;
  debug: boolean;
}

class Application {
  private repo: UserRepository;
  private logger: Logger;
  private config: Config;

  constructor(config: Config) {
    this.config = config;
    this.logger = new Logger(config.debug);
    this.repo = new UserRepository(config.dbUrl);
  }

  async start(): Promise<void> {
    this.logger.info(`Starting application on port ${this.config.port}`);
    await this.repo.connect();
  }

  async stop(): Promise<void> {
    this.logger.info('Shutting down');
    await this.repo.disconnect();
  }

  getConfig(): Config {
    return { ...this.config };
  }
}

export function createApp(config: Config): Application {
  return new Application(config);
}

export { Config, Application };
