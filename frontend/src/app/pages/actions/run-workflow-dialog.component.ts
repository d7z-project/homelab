import { Component, Inject, OnInit, inject } from '@angular/core';
import { CommonModule } from '@angular/common';
import {
  FormsModule,
  ReactiveFormsModule,
  FormBuilder,
  FormGroup,
  Validators,
  ValidatorFn,
  AbstractControl,
  ValidationErrors,
} from '@angular/forms';
import { MatDialogModule, MatDialogRef, MAT_DIALOG_DATA } from '@angular/material/dialog';
import { MatFormFieldModule } from '@angular/material/form-field';
import { MatInputModule } from '@angular/material/input';
import { MatButtonModule } from '@angular/material/button';
import { ModelsWorkflow } from '../../generated';

@Component({
  selector: 'app-run-workflow-dialog',
  standalone: true,
  imports: [
    CommonModule,
    FormsModule,
    ReactiveFormsModule,
    MatDialogModule,
    MatFormFieldModule,
    MatInputModule,
    MatButtonModule,
  ],
  template: `
    <h2 mat-dialog-title>手动运行工作流: {{ data.workflow.name }}</h2>
    <mat-dialog-content>
      <p class="text-sm text-outline mb-4">此工作流需要以下运行参数：</p>
      <form [formGroup]="form" class="flex flex-col gap-4 mt-2">
        @for (key of varKeys; track key) {
          <mat-form-field appearance="outline" class="w-full">
            <mat-label>{{ key }}</mat-label>
            <input
              matInput
              [formControlName]="key"
              [placeholder]="data.workflow.vars?.[key]?.default || ''"
            />
            <mat-hint>{{ data.workflow.vars?.[key]?.description }}</mat-hint>
            @if (form.get(key)?.errors?.['regexMatch']) {
              <mat-error>值不符合该参数的前端正则要求</mat-error>
            }
          </mat-form-field>
        }
      </form>
    </mat-dialog-content>
    <mat-dialog-actions align="end">
      <button mat-button mat-dialog-close>取消</button>
      <button mat-flat-button color="primary" [disabled]="form.valid!" (click)="submit()">
        立即运行
      </button>
    </mat-dialog-actions>
  `,
})
export class RunWorkflowDialogComponent implements OnInit {
  private fb = inject(FormBuilder);
  form!: FormGroup;
  varKeys: string[] = [];

  constructor(
    public dialogRef: MatDialogRef<RunWorkflowDialogComponent>,
    @Inject(MAT_DIALOG_DATA) public data: { workflow: ModelsWorkflow },
  ) {}

  ngOnInit() {
    const group: any = {};
    if (this.data.workflow.vars) {
      this.varKeys = Object.keys(this.data.workflow.vars);
      for (const key of this.varKeys) {
        const def = this.data.workflow.vars[key];
        const validators: any[] = def.required ? [Validators.required] : [];
        if (def.regexFrontend) {
          validators.push((control: AbstractControl): ValidationErrors | null => {
            const val = control.value;
            if (!val || val.includes('${{')) return null;
            try {
              const regex = new RegExp(def.regexFrontend!);
              if (!regex.test(val)) return { regexMatch: true };
            } catch (e) {
              return null;
            }
            return null;
          });
        }
        group[key] = [def.default || '', validators];
      }
    }
    this.form = this.fb.group(group);
  }

  submit() {
    if (this.form.valid) {
      this.dialogRef.close(this.form.value);
    }
  }
}
