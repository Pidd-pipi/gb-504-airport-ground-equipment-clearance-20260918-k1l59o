import { Routes } from '@angular/router';
import { MainLayoutComponent } from '../app/layouts/main-layout.component';
import { authGuard, roleGuard } from '../app/guards';
import { ROLE } from '../constants/enums';

export const routes: Routes = [
  { path: 'login', loadComponent: () => import('../pages/login.page').then(m => m.LoginPage) },
  {
    path: '',
    component: MainLayoutComponent,
    canActivate: [authGuard],
    children: [
      { path: '', pathMatch: 'full', redirectTo: 'turnarounds' },
      { path: 'turnarounds', loadComponent: () => import('../pages/turnarounds.page').then(m => m.TurnaroundsPage) },
      { path: 'ground-units', loadComponent: () => import('../pages/ground-units.page').then(m => m.GroundUnitsPage) },
      { path: 'checks', loadComponent: () => import('../pages/checks.page').then(m => m.ChecksPage) },
      { path: 'clearance', loadComponent: () => import('../pages/clearance.page').then(m => m.ClearancePage) },
      {
        path: 'audit',
        canActivate: [roleGuard(ROLE.ADMIN, ROLE.SAFETY_MANAGER)],
        loadComponent: () => import('../pages/audit.page').then(m => m.AuditPage),
      },
    ],
  },
  { path: '**', redirectTo: 'turnarounds' },
];
