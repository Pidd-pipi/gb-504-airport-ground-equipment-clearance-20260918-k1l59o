import { inject } from '@angular/core';
import { AuthStore } from '../stores/auth.store';

// Reusable auth hook for route-aware components and permission-gated actions.
export function useAuth(): AuthStore {
  return inject(AuthStore);
}
