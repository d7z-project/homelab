import { Component, Inject, OnInit, inject, ChangeDetectorRef } from '@angular/core';
import { CommonModule } from '@angular/common';
import { FormsModule, ReactiveFormsModule, FormBuilder, FormGroup } from '@angular/forms';
import { MatDialogModule, MatDialogRef, MAT_DIALOG_DATA } from '@angular/material/dialog';
import { MatFormFieldModule } from '@angular/material/form-field';
import { MatInputModule } from '@angular/material/input';
import { MatButtonModule } from '@angular/material/button';
import { MatIconModule } from '@angular/material/icon';
import { MatSnackBar } from '@angular/material/snack-bar';
import { ActionsService } from '../../generated';
import { firstValueFrom } from 'rxjs';

@Component({
  selector: 'app-variable-config-dialog',
  standalone: true,
  imports: [
    CommonModule,
    FormsModule,
    ReactiveFormsModule,
    MatDialogModule,
    MatFormFieldModule,
    MatInputModule,
    MatButtonModule,
    MatIconModule,
  ],
  template: `
    <div class="flex flex-col bg-surface overflow-hidden rounded-2xl">
      <header
        class="px-6 py-4 border-b border-outline-variant/30 flex justify-between items-center shrink-0"
      >
        <h2 class="text-lg font-bold m-0">正则表达式校验配置</h2>
        <button mat-icon-button mat-dialog-close><mat-icon>close</mat-icon></button>
      </header>

      <mat-dialog-content class="p-6 m-0!">
        <form [formGroup]="form" class="flex flex-col gap-6 pt-2">
          <p class="text-sm text-outline mb-2">
            为该变量设置校验规则。前端正则用于 UI 实时反馈，后端正则用于执行前最终校验。
          </p>

          <mat-form-field appearance="outline" class="w-full">
            <mat-label>前端校验正则 (Javascript)</mat-label>
            <input matInput formControlName="regexFrontend" placeholder="例如：^[a-zA-Z0-9]+$" />
            <mat-hint>保存工作流时立即触发校验</mat-hint>
            @if (form.get('regexFrontend')?.errors?.['regexInvalid']) {
              <mat-error>{{ form.get('regexFrontend')?.errors?.['regexInvalid'] }}</mat-error>
            }
          </mat-form-field>

          <mat-form-field appearance="outline" class="w-full">
            <mat-label>后端校验正则 (Golang)</mat-label>
            <input matInput formControlName="regexBackend" placeholder="例如：^[a-zA-Z0-9]+$" />
            <mat-hint>触发执行任务时强制进行校验</mat-hint>
            @if (form.get('regexBackend')?.errors?.['regexInvalid']) {
              <mat-error>{{ form.get('regexBackend')?.errors?.['regexInvalid'] }}</mat-error>
            }
          </mat-form-field>
        </form>
      </mat-dialog-content>

      <mat-dialog-actions align="end" class="px-6 py-4 border-t border-outline-variant/10 m-0!">
        <button mat-button mat-dialog-close>取消</button>
        <button mat-flat-button color="primary" [disabled]="loading" (click)="save()">
          {{ loading ? '校验中...' : '确定并保存' }}
        </button>
      </mat-dialog-actions>
    </div>
  `,
})
export class VariableConfigDialogComponent implements OnInit {
  private fb = inject(FormBuilder);
  private orchService = inject(ActionsService);
  private snackBar = inject(MatSnackBar);
  private cdr = inject(ChangeDetectorRef);

  form!: FormGroup;
  loading = false;

  constructor(
    public dialogRef: MatDialogRef<VariableConfigDialogComponent>,
    @Inject(MAT_DIALOG_DATA) public data: { regexFrontend: string; regexBackend: string },
  ) {}

  ngOnInit() {
    this.form = this.fb.group({
      regexFrontend: [this.data.regexFrontend || ''],
      regexBackend: [this.data.regexBackend || ''],
    });
  }

  async save() {
    const { regexFrontend, regexBackend } = this.form.value;
    this.loading = true;

    // Reset errors
    this.form.get('regexFrontend')?.setErrors(null);
    this.form.get('regexBackend')?.setErrors(null);
    this.cdr.detectChanges();

    try {
      if (regexFrontend) {
        try {
          new RegExp(regexFrontend);
        } catch (e: any) {
          this.form
            .get('regexFrontend')
            ?.setErrors({ regexInvalid: 'JS 正则语法错误: ' + e.message });
          throw e;
        }

        try {
          await firstValueFrom(this.orchService.actionsValidateRegexPost(regexFrontend));
        } catch (e: any) {
          this.form
            .get('regexFrontend')
            ?.setErrors({ regexInvalid: e.error?.message || '后端校验失败' });
          throw e;
        }
      }

      if (regexBackend) {
        try {
          await firstValueFrom(this.orchService.actionsValidateRegexPost(regexBackend));
        } catch (e: any) {
          this.form
            .get('regexBackend')
            ?.setErrors({ regexInvalid: e.error?.message || '后端校验失败' });
          throw e;
        }
      }

      this.dialogRef.close(this.form.value);
    } catch (e: any) {
      // Errors are already set on controls
    } finally {
      this.loading = false;
      this.cdr.detectChanges();
    }
  }
}
