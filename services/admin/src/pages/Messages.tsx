import { useEffect, useState } from 'react';
import { useNavigate } from 'react-router-dom';
import {
  Container,
  Paper,
  Table,
  TableBody,
  TableCell,
  TableContainer,
  TableHead,
  TableRow,
  Typography,
  Box,
  Button,
  CircularProgress,
  Alert,
  TextField,
  Stack,
} from '@mui/material';
import { messageApi } from '../api/client';
import { useAuth } from '../context/AuthContext';
import type { Message } from '../types';

export default function Messages() {
  const [messages, setMessages] = useState<Message[]>([]);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string>('');
  const [success, setSuccess] = useState<string>('');
  const [messageText, setMessageText] = useState<string>('');
  const [sending, setSending] = useState(false);
  const { logout, student } = useAuth();
  const navigate = useNavigate();

  useEffect(() => {
    if (student?.email) {
      fetchMessages();
    }
  }, [student]);

  const fetchMessages = async () => {
    if (!student?.email) return;

    setLoading(true);
    setError('');

    try {
      const data = await messageApi.getMessagesByEmail(student.email);
      setMessages(data);
    } catch (err: any) {
      setError(err.response?.data?.error || 'Failed to fetch messages');
    } finally {
      setLoading(false);
    }
  };

  const handleSendMessage = async (e: React.FormEvent) => {
    e.preventDefault();

    if (!messageText.trim()) {
      setError('Message cannot be empty');
      return;
    }

    setSending(true);
    setError('');
    setSuccess('');

    try {
      await messageApi.sendMessage({ message: messageText });
      setSuccess('Message sent successfully!');
      setMessageText('');

      setTimeout(() => {
        fetchMessages();
      }, 1000);
    } catch (err: any) {
      setError(err.response?.data?.error || 'Failed to send message');
    } finally {
      setSending(false);
    }
  };

  const handleLogout = () => {
    logout();
    navigate('/login');
  };

  const formatDate = (dateString: string) => {
    return new Date(dateString).toLocaleString();
  };

  return (
    <Container maxWidth="lg">
      <Box sx={{ mt: 4, mb: 4 }}>
        <Box sx={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', mb: 3 }}>
          <Box>
            <Typography variant="h4" component="h1" gutterBottom>
              Messages
            </Typography>
            {student && (
              <Typography variant="body2" color="text.secondary">
                Logged in as: {student.firstName} {student.lastName} ({student.email})
              </Typography>
            )}
          </Box>
          <Box sx={{ display: 'flex', gap: 2 }}>
            <Button variant="outlined" onClick={() => navigate('/students')}>
              Students
            </Button>
            <Button variant="outlined" color="secondary" onClick={handleLogout}>
              Logout
            </Button>
          </Box>
        </Box>

        <Paper sx={{ p: 3, mb: 3 }}>
          <Typography variant="h6" gutterBottom>
            Send a Message
          </Typography>
          <form onSubmit={handleSendMessage}>
            <Stack direction={{ xs: 'column', sm: 'row' }} spacing={2} alignItems="stretch">
              <Box sx={{ flex: 1 }}>
                <TextField
                  fullWidth
                  label="Message"
                  variant="outlined"
                  value={messageText}
                  onChange={(e) => setMessageText(e.target.value)}
                  disabled={sending}
                  multiline
                  rows={2}
                  placeholder="Type your message here..."
                />
              </Box>
              <Box sx={{ width: { xs: '100%', sm: '150px' } }}>
                <Button
                  fullWidth
                  variant="contained"
                  color="primary"
                  type="submit"
                  disabled={sending || !messageText.trim()}
                  sx={{ height: '100%', minHeight: '56px' }}
                >
                  {sending ? <CircularProgress size={24} /> : 'Send'}
                </Button>
              </Box>
            </Stack>
          </form>
        </Paper>

        {error && (
          <Alert severity="error" sx={{ mb: 2 }} onClose={() => setError('')}>
            {error}
          </Alert>
        )}

        {success && (
          <Alert severity="success" sx={{ mb: 2 }} onClose={() => setSuccess('')}>
            {success}
          </Alert>
        )}

        <Typography variant="h6" gutterBottom sx={{ mt: 3 }}>
          My Messages
        </Typography>

        {loading ? (
          <Box sx={{ display: 'flex', justifyContent: 'center', mt: 4 }}>
            <CircularProgress />
          </Box>
        ) : (
          <TableContainer component={Paper}>
            <Table>
              <TableHead>
                <TableRow>
                  <TableCell>ID</TableCell>
                  <TableCell>Message</TableCell>
                  <TableCell>Sent At</TableCell>
                </TableRow>
              </TableHead>
              <TableBody>
                {messages.length === 0 ? (
                  <TableRow>
                    <TableCell colSpan={3} align="center">
                      No messages found
                    </TableCell>
                  </TableRow>
                ) : (
                  messages.map((msg) => (
                    <TableRow key={msg.id}>
                      <TableCell>{msg.id}</TableCell>
                      <TableCell>{msg.message}</TableCell>
                      <TableCell>{formatDate(msg.createdAt)}</TableCell>
                    </TableRow>
                  ))
                )}
              </TableBody>
            </Table>
          </TableContainer>
        )}
      </Box>
    </Container>
  );
}
