import axios from 'axios';
import type { LoginRequest, AuthResponse, Student, Message, SendMessageRequest } from '../types';

const API_BASE_URL = import.meta.env.VITE_API_URL || 'http://localhost:9080';

const apiClient = axios.create({
  baseURL: API_BASE_URL,
  headers: {
    'Content-Type': 'application/json',
  },
  withCredentials: true,
});

apiClient.interceptors.request.use((config) => {
  const token = localStorage.getItem('accessToken');
  if (token) {
    config.headers.Authorization = `Bearer ${token}`;
  }
  return config;
});

export const authApi = {
  login: async (credentials: LoginRequest): Promise<AuthResponse> => {
    const response = await apiClient.post<AuthResponse>('/auth/login', credentials);
    return response.data;
  },

  logout: async (refreshToken: string): Promise<void> => {
    await apiClient.post('/auth/logout', { refreshToken });
  },
};

export const studentApi = {
  getAllStudents: async (): Promise<Student[]> => {
    const response = await apiClient.get<Student[]>('/api/students');
    return response.data;
  },
};

export const messageApi = {
  sendMessage: async (data: SendMessageRequest): Promise<{ status: string; message: string }> => {
    const response = await apiClient.post('/api/messages', data);
    return response.data;
  },

  getMessagesByEmail: async (email: string): Promise<Message[]> => {
    const response = await apiClient.get<Message[]>(`/api/messages?email=${encodeURIComponent(email)}`);
    return response.data;
  },
};

export default apiClient;
