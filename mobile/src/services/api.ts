import axios, { AxiosInstance, AxiosRequestConfig } from 'axios';
import { store } from '../store/store';

const API_BASE_URL = 'http://localhost:8080/api/v1';

class API {
  private client: AxiosInstance;

  constructor() {
    this.client = axios.create({
      baseURL: API_BASE_URL,
      timeout: 10000,
      headers: {
        'Content-Type': 'application/json',
      },
    });

    this.setupInterceptors();
  }

  private setupInterceptors() {
    // Request interceptor to add auth token
    this.client.interceptors.request.use(
      (config) => {
        const state = store.getState();
        const token = state.auth.accessToken;
        
        if (token) {
          config.headers.Authorization = `Bearer ${token}`;
        }
        
        return config;
      },
      (error) => {
        return Promise.reject(error);
      }
    );

    // Response interceptor to handle token refresh
    this.client.interceptors.response.use(
      (response) => response,
      async (error) => {
        const originalRequest = error.config;

        if (error.response?.status === 401 && !originalRequest._retry) {
          originalRequest._retry = true;

          const state = store.getState();
          const refreshToken = state.auth.refreshToken;

          if (refreshToken) {
            try {
              const response = await this.client.post('/auth/refresh', {
                refresh_token: refreshToken,
              });

              const { access_token, refresh_token } = response.data;
              
              // Update tokens in store
              store.dispatch({
                type: 'auth/setCredentials',
                payload: {
                  user: state.auth.user,
                  accessToken: access_token,
                  refreshToken: refresh_token,
                },
              });

              // Retry original request
              originalRequest.headers.Authorization = `Bearer ${access_token}`;
              return this.client(originalRequest);
            } catch (refreshError) {
              // Refresh failed, logout user
              store.dispatch({ type: 'auth/clearCredentials' });
              return Promise.reject(refreshError);
            }
          }
        }

        return Promise.reject(error);
      }
    );
  }

  // Auth endpoints
  async login(credentials: { email: string; password: string }) {
    return this.client.post('/auth/login', credentials);
  }

  async register(userData: {
    email: string;
    phone?: string;
    password: string;
    first_name: string;
    last_name: string;
    date_of_birth: string;
    gender: string;
  }) {
    return this.client.post('/auth/register', userData);
  }

  async verifyOTP(otpData: { email: string; code: string }) {
    return this.client.post('/auth/verify-otp', otpData);
  }

  async resendOTP(email: string) {
    return this.client.post('/auth/resend-otp', { email });
  }

  async refreshToken(refreshToken: string) {
    return this.client.post('/auth/refresh', { refresh_token: refreshToken });
  }

  async logout() {
    return this.client.post('/auth/logout');
  }

  // User endpoints
  async getProfile() {
    return this.client.get('/users/profile');
  }

  async updateProfile(profileData: any) {
    return this.client.put('/users/profile', profileData);
  }

  async uploadPhoto(photo: FormData) {
    return this.client.post('/users/profile/photo', photo, {
      headers: {
        'Content-Type': 'multipart/form-data',
      },
    });
  }

  async deletePhoto(photoId: number) {
    return this.client.delete(`/users/profile/photo/${photoId}`);
  }

  async discoverUsers(filters: any) {
    return this.client.post('/users/discover', filters);
  }

  async getFavorites() {
    return this.client.get('/users/favorites');
  }

  async addToFavorites(userId: number) {
    return this.client.post(`/users/favorites/${userId}`);
  }

  async removeFromFavorites(userId: number) {
    return this.client.delete(`/users/favorites/${userId}`);
  }

  async blockUser(userId: number) {
    return this.client.post(`/users/block/${userId}`);
  }

  async unblockUser(userId: number) {
    return this.client.delete(`/users/block/${userId}`);
  }

  async reportUser(reportData: {
    reported_id: number;
    reason: string;
    description?: string;
  }) {
    return this.client.post('/users/report', reportData);
  }

  // Match endpoints
  async likeUser(userId: number) {
    return this.client.post(`/matches/like/${userId}`);
  }

  async dislikeUser(userId: number) {
    return this.client.post(`/matches/dislike/${userId}`);
  }

  async getMatches() {
    return this.client.get('/matches');
  }

  async unmatch(matchId: number) {
    return this.client.delete(`/matches/${matchId}`);
  }

  // Message endpoints
  async getConversations() {
    return this.client.get('/messages/conversations');
  }

  async getMessages(conversationId: number) {
    return this.client.get(`/messages/conversations/${conversationId}`);
  }

  async sendMessage(conversationId: number, messageData: {
    content: string;
    message_type?: string;
  }) {
    return this.client.post(`/messages/conversations/${conversationId}`, messageData);
  }

  async markAsRead(conversationId: number) {
    return this.client.put(`/messages/conversations/${conversationId}/read`);
  }

  // WebSocket connection
  getWebSocketURL() {
    const token = store.getState().auth.accessToken;
    return `ws://localhost:8080/api/v1/ws?token=${token}`;
  }
}

export const api = new API();

// Export individual API modules for better organization
export const authAPI = {
  login: api.login.bind(api),
  register: api.register.bind(api),
  verifyOTP: api.verifyOTP.bind(api),
  resendOTP: api.resendOTP.bind(api),
  refreshToken: api.refreshToken.bind(api),
  logout: api.logout.bind(api),
};

export const userAPI = {
  getProfile: api.getProfile.bind(api),
  updateProfile: api.updateProfile.bind(api),
  uploadPhoto: api.uploadPhoto.bind(api),
  deletePhoto: api.deletePhoto.bind(api),
  discoverUsers: api.discoverUsers.bind(api),
  getFavorites: api.getFavorites.bind(api),
  addToFavorites: api.addToFavorites.bind(api),
  removeFromFavorites: api.removeFromFavorites.bind(api),
  blockUser: api.blockUser.bind(api),
  unblockUser: api.unblockUser.bind(api),
  reportUser: api.reportUser.bind(api),
};

export const matchAPI = {
  likeUser: api.likeUser.bind(api),
  dislikeUser: api.dislikeUser.bind(api),
  getMatches: api.getMatches.bind(api),
  unmatch: api.unmatch.bind(api),
};

export const messageAPI = {
  getConversations: api.getConversations.bind(api),
  getMessages: api.getMessages.bind(api),
  sendMessage: api.sendMessage.bind(api),
  markAsRead: api.markAsRead.bind(api),
};

export const websocketAPI = {
  getWebSocketURL: api.getWebSocketURL.bind(api),
};
