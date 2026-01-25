import { createContext, useContext, useState, useEffect, type ReactNode } from 'react';
import type { Student } from '../types';

interface AuthContextType {
  student: Student | null;
  refreshToken: string | null;
  login: (accessToken: string, refreshToken: string, student: Student) => void;
  logout: () => void;
  isAuthenticated: boolean;
}

const AuthContext = createContext<AuthContextType | undefined>(undefined);

export const AuthProvider = ({ children }: { children: ReactNode }) => {
  const [student, setStudent] = useState<Student | null>(null);
  const [refreshToken, setRefreshToken] = useState<string | null>(null);
  const [isAuthenticated, setIsAuthenticated] = useState<boolean>(false);

  useEffect(() => {
    // Check if we have stored refresh token and student
    const storedRefreshToken = localStorage.getItem('refreshToken');
    const storedStudent = localStorage.getItem('student');

    if (storedRefreshToken && storedStudent) {
      setRefreshToken(storedRefreshToken);
      setStudent(JSON.parse(storedStudent));
      setIsAuthenticated(true);
    }
  }, []);

  const login = (_accessToken: string, newRefreshToken: string, newStudent: Student) => {
    // Access token is stored in HttpOnly cookie by backend
    // We only store refresh token and student data
    setRefreshToken(newRefreshToken);
    setStudent(newStudent);
    setIsAuthenticated(true);
    localStorage.setItem('refreshToken', newRefreshToken);
    localStorage.setItem('student', JSON.stringify(newStudent));
  };

  const logout = () => {
    setRefreshToken(null);
    setStudent(null);
    setIsAuthenticated(false);
    localStorage.removeItem('refreshToken');
    localStorage.removeItem('student');
  };

  return (
    <AuthContext.Provider
      value={{
        student,
        refreshToken,
        login,
        logout,
        isAuthenticated,
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
