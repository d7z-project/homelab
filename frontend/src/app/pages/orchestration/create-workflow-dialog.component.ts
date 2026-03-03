import { Component, Inject, OnInit, inject, signal } from '@angular/core';
import { CommonModule } from '@angular/common';
import { FormsModule, ReactiveFormsModule, FormBuilder, FormGroup, Validators, FormArray } from '@angular/forms';
import { MatDialogModule, MatDialogRef, MAT_DIALOG_DATA } from '@angular/material/dialog';
import { MatFormFieldModule } from '@angular/material/form-field';
import { MatInputModule } from '@angular/material/input';
import { MatButtonModule } from '@angular/material/button';
import { MatSelectModule } from '@angular/material/select';
import { MatIconModule } from '@angular/material/icon';
import { MatStepperModule } from '@angular/material/stepper';
import { MatDividerModule } from '@angular/material/divider';
import { MatAutocompleteModule } from '@angular/material/autocomplete';
import { MatToolbarModule } from '@angular/material/toolbar';
import { MatTooltipModule } from '@angular/material/tooltip';
import { OrchestrationService, ModelsWorkflow, ModelsStep, ModelsStepManifest } from '../../generated';
import { firstValueFrom } from 'rxjs';

@Component({
  selector: 'app-create-workflow-dialog',
  standalone: true,
  imports: [
    CommonModule,
    FormsModule,
    ReactiveFormsModule,
    MatDialogModule,
    MatFormFieldModule,
    MatInputModule,
    MatButtonModule,
    MatSelectModule,
    MatIconModule,
    MatStepperModule,
    MatDividerModule,
    MatAutocompleteModule,
    MatToolbarModule,
    MatTooltipModule,
  ],
  template: `
    <div class="flex flex-col h-full bg-surface-container-lowest overflow-hidden">
      <mat-toolbar class="!bg-surface !border-b !border-outline-variant/30 flex justify-between shrink-0 h-16 sm:h-16">
        <div class="flex items-center">
          <button mat-icon-button (click)="dialogRef.close()" matTooltip="返回">
            <mat-icon>close</mat-icon>
          </button>
          <span class="ml-2 text-lg font-medium tracking-tight">{{ data.workflow ? '编辑工作流' : '创建新工作流' }}</span>
        </div>
        <button mat-button color="primary" (click)="submit()" [disabled]="!infoForm.valid || steps.length === 0">
          保存
        </button>
      </mat-toolbar>

      <div class="flex-1 overflow-y-auto p-4 sm:p-8">
        <div class="max-w-4xl mx-auto">
          <mat-stepper orientation="vertical" #stepper class="!bg-transparent">
            <!-- Step 1: Basic Info -->
            <mat-step [stepControl]="infoForm">
              <ng-template matStepLabel>基本信息</ng-template>
              <form [formGroup]="infoForm" class="flex flex-col gap-4 mt-6 max-w-2xl">
                <mat-form-field appearance="outline">
                  <mat-label>工作流名称</mat-label>
                  <input matInput formControlName="name" placeholder="例如：每日数据备份">
                  <mat-error>名称必填</mat-error>
                </mat-form-field>
                <mat-form-field appearance="outline">
                  <mat-label>描述</mat-label>
                  <textarea matInput formControlName="description" placeholder="对该工作流的简要说明" rows="3"></textarea>
                </mat-form-field>
                <div class="mt-2 flex gap-2">
                  <button mat-flat-button matStepperNext type="button" color="primary">下一步</button>
                </div>
              </form>
            </mat-step>

            <!-- Step 2: Configure Steps -->
            <mat-step>
              <ng-template matStepLabel>任务步骤配置</ng-template>
              <div class="flex flex-col gap-6 mt-6 pb-12">
                @for (step of steps.controls; track $index) {
                  <div class="bg-surface border border-outline-variant rounded-2xl overflow-hidden shadow-sm transition-shadow hover:shadow-md relative">
                    <!-- Step Header/Toolbar -->
                    <div class="bg-surface-container-low px-6 py-3 flex justify-between items-center border-b border-outline-variant/30">
                        <div class="flex items-center gap-3">
                            <div class="w-6 h-6 rounded-full bg-primary text-on-primary flex items-center justify-center text-xs font-bold shadow-sm">
                                {{ $index + 1 }}
                            </div>
                            <span class="text-sm font-bold opacity-80">{{ getStepGroup($index).get('name')?.value || '未命名步骤' }}</span>
                        </div>
                        <button mat-icon-button color="warn" (click)="removeStep($index)" matTooltip="删除此步骤" class="!w-8 !h-8">
                            <mat-icon class="!text-lg">delete_outline</mat-icon>
                        </button>
                    </div>

                    <div [formGroup]="getStepGroup($index)" class="p-6 flex flex-col gap-4">
                      <div class="grid grid-cols-1 sm:grid-cols-2 gap-4">
                        <mat-form-field appearance="outline">
                          <mat-label>步骤 ID</mat-label>
                          <input matInput formControlName="id" placeholder="例如：fetch_data">
                          <mat-hint>用于 {{ '{{' }} steps.ID.outputs.key {{ '}}' }} 引用</mat-hint>
                        </mat-form-field>
                        <mat-form-field appearance="outline">
                          <mat-label>显示名称</mat-label>
                          <input matInput formControlName="name" placeholder="例如：获取数据">
                        </mat-form-field>
                      </div>

                      <mat-form-field appearance="outline">
                        <mat-label>处理器类型</mat-label>
                        <mat-select formControlName="type" (selectionChange)="onProcessorChange($index)">
                          @for (m of manifests(); track m.id) {
                            <mat-option [value]="m.id">
                                <div class="flex flex-col">
                                    <span class="font-medium text-sm">{{ m.name }}</span>
                                    <span class="text-[10px] opacity-50">{{ m.id }}</span>
                                </div>
                            </mat-option>
                          }
                        </mat-select>
                      </mat-form-field>

                      <!-- Dynamic Parameters -->
                      @if (getProcessorParams(getStepGroup($index).get('type')?.value).length > 0) {
                        <div class="mt-2 space-y-4">
                            <p class="text-[10px] font-bold uppercase tracking-wider text-outline px-1">输入参数配置</p>
                            <div formGroupName="params" class="grid grid-cols-1 sm:grid-cols-2 gap-x-6 gap-y-4">
                              @for (paramName of getProcessorParams(getStepGroup($index).get('type')?.value); track paramName) {
                                <mat-form-field appearance="outline">
                                  <mat-label>{{ paramName }}</mat-label>
                                  <input matInput [formControlName]="paramName" [matAutocomplete]="auto">
                                  <mat-hint>支持变量引用</mat-hint>
                                  <mat-autocomplete #auto="matAutocomplete">
                                    @for (ref of getOutputReferences($index); track ref) {
                                      <mat-option [value]="ref">{{ ref }}</mat-option>
                                    }
                                  </mat-autocomplete>
                                </mat-form-field>
                              }
                            </div>
                        </div>
                      }
                    </div>
                  </div>
                }

                <div class="flex justify-center py-2">
                    <button mat-stroked-button color="primary" (click)="addStep()" type="button" class="!px-8 border-dashed border-2 inline-flex items-center gap-2">
                      <mat-icon class="!m-0">add</mat-icon>
                      <span>添加一个新的步骤</span>
                    </button>
                </div>

                <div class="mt-6 flex gap-2">
                  <button mat-button matStepperPrevious type="button">上一步</button>
                  <button mat-flat-button matStepperNext type="button" color="primary">预览配置</button>
                </div>
              </div>
            </mat-step>

            <!-- Step 3: Review -->
            <mat-step>
              <ng-template matStepLabel>确认保存</ng-template>
              <div class="mt-8 p-8 bg-surface border border-outline-variant rounded-2xl text-center max-w-2xl mx-auto">
                <mat-icon class="text-5xl h-auto w-auto text-primary opacity-20 mb-4">verified</mat-icon>
                <h3 class="text-xl font-bold mb-2">准备就绪</h3>
                <p class="text-on-surface-variant mb-8 text-sm">
                    工作流 <strong>{{ infoForm.value.name }}</strong> 已配置完成，包含 {{ steps.length }} 个任务步骤。
                    点击顶部的“保存”按钮即可完成操作。
                </p>
                <div class="flex justify-center gap-4">
                    <button mat-button matStepperPrevious type="button">返回修改</button>
                    <button mat-flat-button color="primary" (click)="submit()" [disabled]="!infoForm.valid || steps.length === 0">立即保存</button>
                </div>
              </div>
            </mat-step>
          </mat-stepper>
        </div>
      </div>
    </div>
  `,
  styles: [`
    :host {
        display: block;
        height: 100vh;
    }
    ::ng-deep .mat-stepper-vertical {
        background: transparent !important;
    }
    ::ng-deep .mat-step-header {
        border-radius: 12px !important;
        margin-bottom: 8px !important;
    }
  `]
})
export class CreateWorkflowDialogComponent implements OnInit {
  private fb = inject(FormBuilder);
  private orchService = inject(OrchestrationService);

  manifests = signal<ModelsStepManifest[]>([]);

  infoForm: FormGroup = this.fb.group({
    name: ['', Validators.required],
    description: [''],
  });

  steps: FormArray = this.fb.array([]);

  constructor(
    public dialogRef: MatDialogRef<CreateWorkflowDialogComponent>,
    @Inject(MAT_DIALOG_DATA) public data: { workflow: ModelsWorkflow | null },
  ) {}

  async ngOnInit() {
    const data = await firstValueFrom(this.orchService.orchestrationManifestsGet());
    this.manifests.set(data || []);

    requestAnimationFrame(() => {
      if (this.data.workflow) {
        this.infoForm.patchValue({
          name: this.data.workflow.name,
          description: this.data.workflow.description,
        });

        if (this.data.workflow.steps) {
          for (const s of this.data.workflow.steps) {
            this.addStep(s);
          }
        }
      } else {
          this.addStep(); // Start with one empty step
      }
    });
  }

  getStepGroup(index: number): FormGroup {
    return this.steps.at(index) as FormGroup;
  }

  addStep(stepData?: ModelsStep) {
    const defaultID = `step_${this.steps.length + 1}`;
    const stepGroup = this.fb.group({
      id: [stepData?.id || defaultID, Validators.required],
      name: [stepData?.name || '', Validators.required],
      type: [stepData?.type || '', Validators.required],
      params: this.fb.group({}),
    });

    this.steps.push(stepGroup);

    if (stepData) {
        this.onProcessorChange(this.steps.length - 1, stepData.params);
    }
  }

  removeStep(index: number) {
    this.steps.removeAt(index);
  }

  onProcessorChange(index: number, initialParams?: any) {
    const stepGroup = this.getStepGroup(index);
    const type = stepGroup.get('type')?.value;
    const manifest = this.manifests().find((m) => m.id === type);
    
    const paramsGroup = stepGroup.get('params') as FormGroup;
    // Clear existing params
    Object.keys(paramsGroup.controls).forEach(key => paramsGroup.removeControl(key));

    if (manifest) {
      const allParams = [...(manifest.requiredParams || []), ...(manifest.optionalParams || [])];
      for (const p of allParams) {
        paramsGroup.addControl(p, this.fb.control(initialParams?.[p] || ''));
      }
    }
  }

  getProcessorParams(type: string | undefined): string[] {
    if (!type) return [];
    const manifest = this.manifests().find((m) => m.id === type);
    if (!manifest) return [];
    return [...(manifest.requiredParams || []), ...(manifest.optionalParams || [])];
  }

  getOutputReferences(currentIndex: number): string[] {
    const refs: string[] = [];
    for (let i = 0; i < currentIndex; i++) {
      const step = this.getStepGroup(i);
      const stepID = step.get('id')?.value;
      const type = step.get('type')?.value;
      const manifest = this.manifests().find((m) => m.id === type);
      
      if (stepID && manifest && manifest.outputParams) {
        for (const op of manifest.outputParams) {
          refs.push(`\${{ steps.${stepID}.outputs.${op} }}`);
        }
      }
    }
    return refs;
  }

  submit() {
    const workflow: ModelsWorkflow = {
      ...this.infoForm.value,
      steps: this.steps.value,
    };
    this.dialogRef.close(workflow);
  }
}
