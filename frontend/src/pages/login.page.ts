import { CommonModule } from '@angular/common';
import { Component, inject } from '@angular/core';
import { FormBuilder, ReactiveFormsModule, Validators } from '@angular/forms';
import { Router } from '@angular/router';
import { MatButtonModule } from '@angular/material/button';
import { MatFormFieldModule } from '@angular/material/form-field';
import { MatIconModule } from '@angular/material/icon';
import { MatInputModule } from '@angular/material/input';
import { MatProgressSpinnerModule } from '@angular/material/progress-spinner';
import { loginApi } from '../api/auth.api';
import { useAuth } from '../hooks/use-auth';
import { parseHttpError, useHttp } from '../utils/request';

@Component({
  selector: 'app-login-page',
  standalone: true,
  imports: [CommonModule, ReactiveFormsModule, MatButtonModule, MatFormFieldModule, MatIconModule, MatInputModule, MatProgressSpinnerModule],
  template: `
    <main class="login-shell">
      <section class="context">
        <div class="mark"><mat-icon>flight_takeoff</mat-icon></div>
        <p class="eyebrow">AIRPORT OPERATIONS</p>
        <h1>机坪地面设备<br>安全放行控制台</h1>
        <p class="summary">统一管理周转阶段、设备状态、现场证据和放行决定。</p>
        <div class="signals"><span><i></i>航班周转在线</span><span><i></i>审计链路在线</span></div>
      </section>
      <section class="login-panel">
        <form [formGroup]="form" (ngSubmit)="submit()">
          <header><p>值班账号</p><h2>登录控制台</h2></header>
          <mat-form-field appearance="outline">
            <mat-label>手机号</mat-label>
            <mat-icon matPrefix>badge</mat-icon>
            <input matInput formControlName="phone" autocomplete="username">
          </mat-form-field>
          <mat-form-field appearance="outline">
            <mat-label>密码</mat-label>
            <mat-icon matPrefix>lock</mat-icon>
            <input matInput type="password" formControlName="password" autocomplete="current-password">
          </mat-form-field>
          <p class="error" *ngIf="error">{{ error }}</p>
          <button mat-flat-button type="submit" [disabled]="form.invalid || loading">
            <mat-spinner *ngIf="loading" diameter="20"></mat-spinner><span *ngIf="!loading">进入作业台</span>
          </button>
          <p class="hint">管理员：13800000001 / Admin&#64;123</p>
        </form>
      </section>
    </main>
  `,
  styles: [`
    .login-shell { min-height: 100vh; display: grid; grid-template-columns: minmax(360px, 1.2fr) minmax(420px, .8fr); background: #18242b; }
    .context { color: #fff; align-self: center; padding: clamp(48px, 8vw, 120px); }
    .mark { width: 48px; height: 48px; display: grid; place-items: center; background: #12a59a; border-radius: 6px; margin-bottom: 44px; }
    .eyebrow { color: #56d1c7; font-size: 12px; letter-spacing: 0; }
    h1 { font-size: clamp(34px, 5vw, 58px); line-height: 1.12; letter-spacing: 0; margin: 14px 0 22px; }
    .summary { color: #a9bbc1; max-width: 480px; font-size: 16px; line-height: 1.7; }
    .signals { display: flex; gap: 26px; margin-top: 54px; color: #b9c8cd; font-size: 12px; }
    .signals span { display: flex; align-items: center; gap: 8px; } .signals i { width: 7px; height: 7px; background: #4cc38a; border-radius: 50%; }
    .login-panel { display: grid; place-items: center; background: #f4f6f6; padding: 32px; }
    form { width: min(100%, 380px); }
    header { margin-bottom: 30px; } header p { color: #0e8e85; font-size: 12px; font-weight: 700; } h2 { margin: 4px 0 0; font-size: 28px; letter-spacing: 0; }
    mat-form-field { width: 100%; margin-bottom: 6px; }
    form > button { width: 100%; height: 46px; background: #0b8f86; color: #fff; border-radius: 5px; }
    .error { color: #b42318; font-size: 12px; margin-top: 0; }
    .hint { color: #728187; text-align: center; font-size: 12px; margin-top: 18px; }
    @media (max-width: 800px) { .login-shell { grid-template-columns: 1fr; background: #f4f6f6; } .context { padding: 34px 24px; background: #18242b; } .context h1 { font-size: 32px; } .summary, .signals { display: none; } .mark { margin-bottom: 20px; } .login-panel { padding: 42px 24px; } }
  `],
})
export class LoginPage {
  private readonly fb = inject(FormBuilder);
  private readonly http = useHttp();
  private readonly auth = useAuth();
  private readonly router = inject(Router);
  readonly form = this.fb.nonNullable.group({
    phone: ['13800000001', [Validators.required]],
    password: ['Admin@123', [Validators.required]],
  });
  loading = false;
  error = '';

  submit(): void {
    if (this.form.invalid) return;
    this.loading = true;
    this.error = '';
    loginApi(this.http, this.form.getRawValue()).subscribe({
      next: response => { this.auth.setAuth(response); void this.router.navigate(['/turnarounds']); },
      error: error => { this.loading = false; this.error = parseHttpError(error); },
    });
  }
}
