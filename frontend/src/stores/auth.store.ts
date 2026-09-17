import { Injectable, computed, signal } from '@angular/core';
import { LoginResponse, User } from '../types';

const TOKEN_KEY = 'ground-clearance-token';
const USER_KEY = 'ground-clearance-user';

function storedUser(): User | null {
  try {
    const raw = localStorage.getItem(USER_KEY);
    return raw ? JSON.parse(raw) as User : null;
  } catch {
    return null;
  }
}

@Injectable({ providedIn: 'root' })
export class AuthStore {
  private readonly tokenState = signal(localStorage.getItem(TOKEN_KEY) ?? '');
  private readonly userState = signal<User | null>(storedUser());

  readonly token = this.tokenState.asReadonly();
  readonly user = this.userState.asReadonly();
  readonly isLoggedIn = computed(() => Boolean(this.tokenState()));

  setAuth(response: LoginResponse): void {
    this.tokenState.set(response.token);
    this.userState.set(response.user);
    localStorage.setItem(TOKEN_KEY, response.token);
    localStorage.setItem(USER_KEY, JSON.stringify(response.user));
  }

  clear(): void {
    this.tokenState.set('');
    this.userState.set(null);
    localStorage.removeItem(TOKEN_KEY);
    localStorage.removeItem(USER_KEY);
  }

  hasRole(...roles: string[]): boolean {
    const user = this.userState();
    return Boolean(user && roles.includes(user.role));
  }
}
