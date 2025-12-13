export interface Student {
  id: number;
  firstName: string;
  lastName: string;
  email: string;
  major: string;
  year: number;
}

export interface LoginRequest {
  email: string;
  password: string;
}

export interface AuthResponse {
  accessToken: string;
  refreshToken: string;
  student: Student;
}

export interface Message {
  id: number;
  email: string;
  message: string;
  createdAt: string;
}

export interface SendMessageRequest {
  message: string;
}
