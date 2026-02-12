import { Component, OnInit, inject, ChangeDetectorRef } from '@angular/core';
import { FormBuilder, FormGroup, Validators, ReactiveFormsModule } from '@angular/forms';
import { Router } from '@angular/router';
import { NzFormModule } from 'ng-zorro-antd/form';
import { NzInputModule } from 'ng-zorro-antd/input';
import { NzButtonModule } from 'ng-zorro-antd/button';
import { NzCardModule } from 'ng-zorro-antd/card';
import { NzIconModule } from 'ng-zorro-antd/icon';
import { CommonModule } from '@angular/common';
import { AuthService } from '../../generated';
import { NzMessageService } from 'ng-zorro-antd/message';
import { firstValueFrom } from 'rxjs';

import { HttpErrorResponse } from '@angular/common/http';

@Component({
  selector: 'app-login',

  standalone: true,

  imports: [
    CommonModule,

    ReactiveFormsModule,

    NzFormModule,

    NzInputModule,

    NzButtonModule,

    NzCardModule,

    NzIconModule,
  ],

  templateUrl: './login.component.html',

  styleUrls: ['./login.component.css'],
})
export class LoginComponent implements OnInit {
  validateForm!: FormGroup;

  loading = false;

  showTotp = false;

  private fb = inject(FormBuilder);

  private router = inject(Router);

  private authService = inject(AuthService);

  private message = inject(NzMessageService);

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

        this.message.success('登录成功');

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

            this.message.info(errorMsg);

            return;
          } else if (apiError.code === 10000) {
            errorMsg = '密码或验证码错误';
          } else if (apiError.message) {
            errorMsg = apiError.message;
          }
        }

        this.message.error(errorMsg);
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
