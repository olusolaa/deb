import axios from 'axios';

// Use environment variable for backend URL, fallback for local dev
const API_BASE_URL = process.env.REACT_APP_API_URL || 'http://localhost:8080';

const apiClient = axios.create({
  baseURL: API_BASE_URL,
  // Tell Axios to send cookies with cross-origin requests
  // This is crucial for the HttpOnly auth_token cookie to be sent
  withCredentials: true,
});

// Optional: Add interceptors for request/response handling if needed
// For example, automatically refreshing tokens or handling specific errors

// apiClient.interceptors.response.use(
//   (response) => response,
//   (error) => {
//     if (error.response && error.response.status === 401) {
//       // Handle unauthorized errors (e.g., redirect to login)
//       console.error('Unauthorized access - redirecting to login');
//       // window.location.href = '/login'; // Or use react-router history
//     }
//     return Promise.reject(error);
//   }
// );

export default apiClient;
