export interface AdminEntity {
  id: string;
  email: string;
  passwordHash: string;
  name: string | null;
  createdAt: string;
}

export interface AdminLoginResult {
  id: string;
  email: string;
  name: string | null;
}
