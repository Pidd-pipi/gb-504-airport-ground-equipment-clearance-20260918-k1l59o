import { CommonModule } from '@angular/common';
import { HttpClient } from '@angular/common/http';
import { Component, inject, OnDestroy, signal } from '@angular/core';
import { BreakpointObserver } from '@angular/cdk/layout';
import { Router, RouterLink, RouterLinkActive, RouterOutlet } from '@angular/router';
import { MatButtonModule } from '@angular/material/button';
import { MatIconModule } from '@angular/material/icon';
import { MatListModule } from '@angular/material/list';
import { MatSidenavModule } from '@angular/material/sidenav';
import { MatToolbarModule } from '@angular/material/toolbar';
import { MatTooltipModule } from '@angular/material/tooltip';
import { ROLE, ROLE_TEXT } from '../../constants/enums';
import { useAuth } from '../../hooks/use-auth';
import { environment } from '../../environments/environment';

@Component({
  selector: 'app-main-layout',
  standalone: true,
  imports: [
    CommonModule, RouterOutlet, RouterLink, RouterLinkActive, MatButtonModule, MatIconModule,
    MatListModule, MatSidenavModule, MatToolbarModule, MatTooltipModule,
  ],
  template: `
    <mat-sidenav-container class="shell">
      <mat-sidenav #drawer [mode]="mobile ? 'over' : 'side'" [opened]="!mobile" class="nav">
        <div class="brand">
          <span class="brand-mark"><mat-icon>flight_takeoff</mat-icon></span>
          <span><strong>机坪放行台</strong><small>Ground Clearance</small></span>
        </div>
        <div class="shift-label">当班作业</div>
        <mat-nav-list>
          <a mat-list-item routerLink="/turnarounds" routerLinkActive="active" (click)="closeMobile(drawer)">
            <mat-icon matListItemIcon>view_kanban</mat-icon><span matListItemTitle>航班周转</span>
          </a>
          <a mat-list-item routerLink="/ground-units" routerLinkActive="active" (click)="closeMobile(drawer)">
            <mat-icon matListItemIcon>airport_shuttle</mat-icon><span matListItemTitle>地面设备</span>
          </a>
          <a mat-list-item routerLink="/checks" routerLinkActive="active" (click)="closeMobile(drawer)">
            <mat-icon matListItemIcon>fact_check</mat-icon><span matListItemTitle>检查工作台</span>
          </a>
          <a mat-list-item routerLink="/clearance" routerLinkActive="active" (click)="closeMobile(drawer)">
            <mat-icon matListItemIcon>verified_user</mat-icon><span matListItemTitle>安全放行</span>
          </a>
          <a mat-list-item *ngIf="auth.hasRole(ROLE.ADMIN, ROLE.SAFETY_MANAGER)" routerLink="/audit" routerLinkActive="active" (click)="closeMobile(drawer)">
            <mat-icon matListItemIcon>history</mat-icon><span matListItemTitle>审计追踪</span>
          </a>
        </mat-nav-list>
        <div class="nav-foot" [class.degraded]="health() === 'degraded'">
          <span class="live-dot"></span>
          <span>{{ health() === 'ok' ? '运行状态正常' : health() === 'checking' ? '正在检查依赖' : '依赖服务异常' }}</span>
        </div>
      </mat-sidenav>
      <mat-sidenav-content>
        <mat-toolbar class="toolbar">
          <button *ngIf="mobile" mat-icon-button aria-label="打开导航" matTooltip="导航" (click)="drawer.toggle()">
            <mat-icon>menu</mat-icon>
          </button>
          <div class="station"><span>机场地勤安全控制中心</span><small>08:00 - 20:00 白班</small></div>
          <span class="spacer"></span>
          <div class="identity">
            <span class="avatar">{{ (auth.user()?.name || 'U').slice(0, 1) }}</span>
            <span><strong>{{ auth.user()?.name }}</strong><small>{{ roleText[auth.user()?.role || ''] || auth.user()?.role }}</small></span>
          </div>
          <button mat-icon-button aria-label="退出登录" matTooltip="退出登录" (click)="logout()"><mat-icon>logout</mat-icon></button>
        </mat-toolbar>
        <main class="workspace"><router-outlet></router-outlet></main>
      </mat-sidenav-content>
    </mat-sidenav-container>
  `,
  styles: [`
    .shell { height: 100vh; background: #f2f4f5; }
    .nav { width: 248px; background: #18242b; color: #eef5f5; border: 0; }
    .brand { height: 72px; display: flex; align-items: center; gap: 12px; padding: 0 20px; border-bottom: 1px solid rgba(255,255,255,.08); }
    .brand-mark { width: 36px; height: 36px; display: grid; place-items: center; background: #12a59a; border-radius: 6px; }
    .brand strong, .brand small, .identity strong, .identity small, .station small { display: block; }
    .brand strong { font-size: 15px; letter-spacing: 0; }
    .brand small { color: #91a5ad; font-size: 10px; margin-top: 2px; }
    .shift-label { padding: 22px 20px 8px; color: #80949c; font-size: 11px; }
    mat-nav-list a { margin: 3px 10px; width: calc(100% - 20px); border-radius: 5px; color: #b9c8cd; }
    mat-nav-list a.active { color: #fff; background: rgba(18,165,154,.2); }
    mat-nav-list mat-icon { color: inherit; }
    .nav-foot { position: absolute; bottom: 20px; left: 20px; right: 20px; display: flex; align-items: center; gap: 8px; color: #91a5ad; font-size: 12px; }
    .live-dot { width: 8px; height: 8px; border-radius: 50%; background: #4cc38a; box-shadow: 0 0 0 3px rgba(76,195,138,.12); }
    .nav-foot.degraded { color: #ffc9c5; } .nav-foot.degraded .live-dot { background: #e2574c; box-shadow: 0 0 0 3px rgba(226,87,76,.16); }
    .toolbar { height: 72px; background: #fff; color: #21313a; border-bottom: 1px solid #dfe5e7; padding: 0 22px; }
    .station { font-size: 15px; font-weight: 600; }
    .station small { color: #7a8b92; font-size: 11px; font-weight: 400; margin-top: 2px; }
    .spacer { flex: 1; }
    .identity { display: flex; align-items: center; gap: 9px; margin-right: 10px; font-size: 12px; }
    .identity small { color: #718188; font-size: 10px; }
    .avatar { width: 32px; height: 32px; display: grid; place-items: center; border-radius: 50%; background: #e8f5f3; color: #087e76; font-weight: 700; }
    .workspace { max-width: 1480px; margin: 0 auto; padding: 22px; }
    @media (max-width: 820px) { .workspace { padding: 14px; } .station small, .identity span:not(.avatar) { display: none; } }
  `],
})
export class MainLayoutComponent implements OnDestroy {
  readonly auth = useAuth();
  readonly ROLE = ROLE;
  readonly roleText = ROLE_TEXT;
  private readonly router = inject(Router);
  private readonly http = inject(HttpClient);
  readonly health = signal<'checking' | 'ok' | 'degraded'>('checking');
  private readonly healthTimer: number;
  mobile = false;

  constructor() {
    inject(BreakpointObserver).observe('(max-width: 820px)').subscribe(result => this.mobile = result.matches);
    this.refreshHealth();
    this.healthTimer = window.setInterval(() => this.refreshHealth(), 30_000);
  }

  ngOnDestroy(): void { window.clearInterval(this.healthTimer); }

  closeMobile(drawer: { close(): Promise<unknown> }): void {
    if (this.mobile) void drawer.close();
  }

  logout(): void {
    this.auth.clear();
    void this.router.navigate(['/login']);
  }

  private refreshHealth(): void {
    this.http.get<{ status: string }>(`${environment.apiUrl}/healthz`).subscribe({
      next: result => this.health.set(result.status === 'ok' ? 'ok' : 'degraded'),
      error: () => this.health.set('degraded'),
    });
  }
}
