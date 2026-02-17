export interface User {
  id: number;
  name: string;
  email: string;
}

export class UserRepository {
  private dbUrl: string;
  private connected: boolean = false;

  constructor(dbUrl: string) {
    this.dbUrl = dbUrl;
  }

  async connect(): Promise<void> {
    this.connected = true;
  }

  async disconnect(): Promise<void> {
    this.connected = false;
  }

  async findById(id: number): Promise<User | null> {
    if (!this.connected) throw new Error('Not connected');
    return null;
  }

  async findAll(): Promise<User[]> {
    if (!this.connected) throw new Error('Not connected');
    return [];
  }

  async save(user: Omit<User, 'id'>): Promise<User> {
    if (!this.connected) throw new Error('Not connected');
    return { id: 1, ...user };
  }
}
