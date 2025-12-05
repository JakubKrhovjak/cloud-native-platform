import { createContext, useContext, useState, useEffect, type ReactNode } from 'react';
import type { Student } from '../types';

interface AuthContextType {
  student: Student | null;
  accessToken: string | null;
  refreshToken: string | null;
  login: (accessToken: string, refreshToken: string, student: Student) => void;
  logout: () => void;
  isAuthenticated: boolean;
}

const AuthContext = createContext<AuthContextType | undefined>(undefined);

export const AuthProvider = ({ children }: { children: ReactNode }) => {
  const [student, setStudent] = useState<Student | null>(null);
  const [accessToken, setAccessToken] = useState<string | null>(null);
  const [refreshToken, setRefreshToken] = useState<string | null>(null);

  useEffect(() => {
    const storedAccessToken = localStorage.getItem('accessToken');
    const storedRefreshToken = localStorage.getItem('refreshToken');
    const storedStudent = localStorage.getItem('student');

    if (storedAccessToken && storedRefreshToken && storedStudent) {
      setAccessToken(storedAccessToken);
      setRefreshToken(storedRefreshToken);
      setStudent(JSON.parse(storedStudent));
    }
  }, []);

  const login = (newAccessToken: string, newRefreshToken: string, newStudent: Student) => {
    setAccessToken(newAccessToken);
    setRefreshToken(newRefreshToken);
    setStudent(newStudent);
    localStorage.setItem('accessToken', newAccessToken);
    localStorage.setItem('refreshToken', newRefreshToken);
    localStorage.setItem('student', JSON.stringify(newStudent));
  };

  const logout = () => {
    setAccessToken(null);
    setRefreshToken(null);
    setStudent(null);
    localStorage.removeItem('accessToken');
    localStorage.removeItem('refreshToken');
    localStorage.removeItem('student');
  };

  return (
    <AuthContext.Provider
      value={{
        student,
        accessToken,
        refreshToken,
        login,
        logout,
        isAuthenticated: !!accessToken,
      }}
    >
      {children}
    </AuthContext.Provider>
  );
};

export const useAuth = () => {
  const context = useContext(AuthContext);
  if (context === undefined) {
    throw new Error('useAuth must be used within an AuthProvider');
  }
  return context;
};
