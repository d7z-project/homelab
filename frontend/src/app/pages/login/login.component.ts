import { Component, OnInit, inject, ChangeDetectorRef } from '@angular/core';
import { FormBuilder, FormGroup, Validators, ReactiveFormsModule } from '@angular/forms';
import { Router } from '@angular/router';
import { MatFormFieldModule } from '@angular/material/form-field';
import { MatInputModule } from '@angular/material/input';
import { MatButtonModule } from '@angular/material/button';
import { MatCardModule } from '@angular/material/card';
import { MatIconModule } from '@angular/material/icon';
import { CommonModule } from '@angular/common';
import { AuthService } from '../../generated';
import { MatSnackBar } from '@angular/material/snack-bar';
import { firstValueFrom } from 'rxjs';

import { HttpErrorResponse } from '@angular/common/http';

@Component({
  selector: 'app-login',

  standalone: true,

  imports: [
    CommonModule,

    ReactiveFormsModule,

    MatFormFieldModule,

    MatInputModule,

    MatButtonModule,

    MatCardModule,

    MatIconModule,
  ],

  templateUrl: './login.component.html',
})
export class LoginComponent implements OnInit {
  validateForm!: FormGroup;

  loading = false;

  showTotp = false;

  private fb = inject(FormBuilder);

  private router = inject(Router);

  private authService = inject(AuthService);

  private snackBar = inject(MatSnackBar);

  private cdr = inject(ChangeDetectorRef);

  ngOnInit(): void {
    this.validateForm = this.fb.group({
      password: ['', [Validators.required]],

      totp: [''],
    });
  }

  async submitForm(): Promise<void> {
    if (this.validateForm.valid) {
      this.loading = true;

      try {
        const res = await firstValueFrom(
          this.authService.loginPost({
            password: this.validateForm.value.password,
            totp: this.validateForm.value.totp,
          }),
        );

        localStorage.setItem('session_id', res.session_id!);
        this.snackBar.open('登录成功', '关闭', { duration: 3000 });
        this.router.navigate(['/']);
      } catch (err) {
        let errorMsg = '登录失败，请检查密码或验证码';

        if (err instanceof HttpErrorResponse && err.error) {
          const apiError = err.error;

          if (apiError.code === 10001) {
            this.showTotp = true;

            this.validateForm.get('totp')?.setValidators([Validators.required]);

            this.validateForm.get('totp')?.updateValueAndValidity();

            errorMsg = '请输入 TOTP 验证码';

            this.snackBar.open(errorMsg, '关闭', { duration: 3000 });

            return;
          } else if (apiError.code === 10000) {
            errorMsg = '密码或验证码错误';
          } else if (apiError.message) {
            errorMsg = apiError.message;
          }
        }

        this.snackBar.open(errorMsg, '关闭', { duration: 3000 });
      } finally {
        this.loading = false;

        this.cdr.detectChanges();
      }
    } else {
      Object.values(this.validateForm.controls).forEach((control) => {
        if (control.invalid) {
          control.markAsDirty();

          control.updateValueAndValidity({ onlySelf: true });
        }
      });
    }
  }
}
