import axios from 'axios';

// Use environment variable for backend URL, dynamically determine fallback for production
const ENV = window.ENV || {};

// Function to determine the backend URL based on current host if env vars not set
const getBackendURLFromHost = () => {
  const currentHost = window.location.host;
  const protocol = window.location.protocol;
  
  // For local development
  if (currentHost.includes('localhost') || currentHost.includes('127.0.0.1')) {
    return 'http://localhost:8084';
  }
  
  // For production (likely Render)
  console.warn(
    'WARNING: API_BASE_URL environment variable not provided.\n' +
    'Please set REACT_APP_API_URL in your environment variables.\n' +
    'Using same-origin API URL fallback - this only works if backend and frontend share the same domain.'
  );
  
  // Default to same origin (only works if your API is on the same domain)
  return window.location.origin;
};

// Use env var if available, otherwise determine from host
const API_BASE_URL = ENV.REACT_APP_API_URL || process.env.REACT_APP_API_URL || getBackendURLFromHost();

// Log the API URL for debugging
console.log('API client configured with base URL:', API_BASE_URL);

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
