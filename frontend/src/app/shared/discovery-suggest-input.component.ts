import { Component, Input, OnInit, inject, signal, Optional, Self } from '@angular/core';
import { CommonModule } from '@angular/common';
import {
  ControlValueAccessor,
  FormsModule,
  ReactiveFormsModule,
  FormControl,
  Validator,
  AbstractControl,
  ValidationErrors,
  NgControl,
} from '@angular/forms';
import { MatFormFieldModule } from '@angular/material/form-field';
import { MatInputModule } from '@angular/material/input';
import {
  MatAutocompleteModule,
  MatAutocompleteSelectedEvent,
} from '@angular/material/autocomplete';
import { MatIconModule } from '@angular/material/icon';
import { MatProgressBarModule } from '@angular/material/progress-bar';
import { DiscoveryService, ModelsLookupItem, ModelsDiscoverResult } from '../generated';
import { debounceTime, distinctUntilChanged, switchMap, of, catchError, finalize } from 'rxjs';

/**
 * AppDiscoverySuggestInputComponent
 * Provides a free-text input with suggestions from:
 * 1. Remote lookup (discovery service)
 * 2. Static string list (e.g. variable references)
 * 3. Structured RBAC resource results (supports drill-down)
 */
@Component({
  selector: 'app-discovery-suggest-input',
  standalone: true,
  imports: [
    CommonModule,
    FormsModule,
    ReactiveFormsModule,
    MatFormFieldModule,
    MatInputModule,
    MatAutocompleteModule,
    MatIconModule,
    MatProgressBarModule,
  ],
  template: `
    <mat-form-field
      [appearance]="appearance"
      class="w-full relative"
      [subscriptSizing]="subscriptSizing"
    >
      <mat-label>{{ label }}</mat-label>
      <input
        matInput
        [placeholder]="placeholder"
        [matAutocomplete]="auto"
        [formControl]="control"
        (blur)="onTouched()"
      />

      <!-- Loading bar at bottom of field -->
      <div
        class="absolute left-0 right-0 bottom-0 overflow-hidden transition-all duration-300 pointer-events-none"
        [style.height]="isLoading() ? '2px' : '0px'"
        [style.opacity]="isLoading() ? 1 : 0"
      >
        <mat-progress-bar mode="indeterminate" class="h-[2px]!"></mat-progress-bar>
      </div>

      <mat-autocomplete #auto="matAutocomplete" (optionSelected)="onSelected($event)">
        <!-- Variable/Ref Suggestions (Priority) -->
        @for (ref of staticSuggestions; track ref) {
          <mat-option [value]="ref" class="h-auto! py-3!">
            <div class="flex items-center gap-4">
              <div
                class="w-10 h-10 rounded-xl bg-secondary/10 flex items-center justify-center flex-shrink-0"
              >
                <mat-icon
                  class="m-0! text-[24px]! w-6! h-6! leading-none! flex items-center justify-center text-secondary/70"
                  >data_object</mat-icon
                >
              </div>
              <span class="font-mono text-sm text-secondary truncate">{{ ref }}</span>
            </div>
          </mat-option>
        }

        <!-- RBAC Discovery Results -->
        @for (ref of rbacSuggestions; track ref.fullId) {
          <mat-option [value]="ref" class="h-auto! py-3!">
            <div class="flex items-start gap-4">
              <div
                class="w-10 h-10 rounded-xl bg-surface-container flex items-center justify-center flex-shrink-0 mt-0.5"
              >
                <mat-icon
                  class="m-0! text-[24px]! w-6! h-6! leading-none! flex items-center justify-center opacity-70"
                  >{{ ref.final ? 'description' : 'folder' }}</mat-icon
                >
              </div>
              <div class="flex flex-col min-w-0 flex-1 leading-tight">
                <span class="font-bold text-[14px] text-on-surface truncate">{{ ref.name }}</span>
                <span class="text-[11px] font-mono text-outline truncate opacity-60 mt-0.5">
                  {{ ref.fullId }}
                </span>
              </div>
            </div>
          </mat-option>
        }

        <!-- Discovery Results -->
        @for (item of items(); track item.id) {
          <mat-option [value]="item.id" class="h-auto! py-3!">
            <div class="flex items-start gap-4">
              <div
                class="w-10 h-10 rounded-xl bg-surface-container flex items-center justify-center flex-shrink-0 mt-0.5"
              >
                <mat-icon
                  class="m-0! text-[24px]! w-6! h-6! leading-none! flex items-center justify-center opacity-70"
                  >{{ item.icon || 'label' }}</mat-icon
                >
              </div>
              <div class="flex flex-col min-w-0 flex-1 leading-tight">
                <span class="font-bold text-[14px] text-on-surface truncate">{{ item.name }}</span>
                <span class="text-[11px] font-mono text-outline truncate opacity-60 mt-0.5">
                  {{ item.id }}
                </span>
                @if (item.description) {
                  <span
                    class="text-[11px] text-outline truncate opacity-80 mt-1 italic leading-tight"
                  >
                    {{ item.description }}
                  </span>
                }
              </div>
            </div>
          </mat-option>
        }
      </mat-autocomplete>

      @if (ngControl && ngControl.errors && ngControl.errors['notFinal']) {
        <mat-error>请选择更具体的资源路径 (当前为目录/分类)</mat-error>
      } @else if (ngControl && ngControl.errors && ngControl.errors['invalidPath']) {
        <mat-error>无效的资源路径 (未找到匹配项)</mat-error>
      } @else if (ngControl && ngControl.errors && ngControl.errors['required']) {
        <mat-error>此项为必填项</mat-error>
      }

      @if (hint && (!ngControl || !ngControl.invalid)) {
        <mat-hint>{{ hint }}</mat-hint>
      }
    </mat-form-field>
  `,
})
export class DiscoverySuggestInputComponent implements OnInit, ControlValueAccessor, Validator {
  private discoveryService = inject(DiscoveryService);

  /** Standard lookup code for discovery service */
  @Input() code = '';
  /** Main field label */
  @Input() label = '输入值';
  /** Field placeholder */
  @Input() placeholder = '输入自定义内容或 ${{ 引用变量 }}';
  /** Optional hint text */
  @Input() hint = '';
  /** Material field appearance */
  @Input() appearance: 'fill' | 'outline' = 'outline';
  /** Subscript sizing behavior */
  @Input() subscriptSizing: 'fixed' | 'dynamic' = 'fixed';

  /** Static string suggestions (e.g. for variable completion) */
  @Input() staticSuggestions: string[] = [];

  /** Structured RBAC suggestions */
  @Input() rbacSuggestions: ModelsDiscoverResult[] = [];

  /** Whether the parent is currently fetching suggestions */
  @Input() loading = false;

  /** Whether to strictly validate against RBAC rules */
  @Input() rbacMode = false;

  control = new FormControl('');
  items = signal<ModelsLookupItem[]>([]);
  isLoading = signal(false);
  disabled = false;

  constructor(@Optional() @Self() public ngControl: NgControl) {
    if (this.ngControl) {
      this.ngControl.valueAccessor = this;
    }
  }

  public onChange: (value: any) => void = () => {};
  public onTouched: () => void = () => {};

  ngOnInit() {
    // Register validator manually since we use @Self() NgControl
    if (this.ngControl && this.ngControl.control) {
      this.ngControl.control.addValidators(this.validate.bind(this));
    }

    this.control.valueChanges
      .pipe(
        debounceTime(300),
        distinctUntilChanged(),
        switchMap((value) => {
          // If value is a ModelsDiscoverResult (from selection), we don't want to trigger lookup
          if (typeof value !== 'string') return of({ items: [] });

          const search = value;
          // Only trigger discovery search if code is provided and value doesn't look like a variable ref
          if (!this.code || search.includes('${{')) {
            return of({ items: [] });
          }

          this.isLoading.set(true);
          return this.discoveryService.discoveryLookupGet(this.code, search, '', 10).pipe(
            catchError(() => of({ items: [] })),
            finalize(() => this.isLoading.set(false)),
          );
        }),
      )
      .subscribe((res) => {
        this.items.set(res.items || []);
      });

    // Notify parent on change
    this.control.valueChanges.subscribe((val) => {
      if (typeof val === 'string') {
        this.onChange(val);
      }
    });
  }

  onSelected(event: MatAutocompleteSelectedEvent) {
    const val = event.option.value;
    if (val && typeof val === 'object' && 'fullId' in val) {
      const result = val as ModelsDiscoverResult;
      let nextValue = result.fullId || '';
      if (!result.final) {
        nextValue += '/';
      }
      this.control.setValue(nextValue);
    } else if (typeof val === 'string') {
      this.control.setValue(val);
    }
  }

  // Validator implementation
  validate(control: AbstractControl): ValidationErrors | null {
    const val = control.value;
    if (!val || typeof val !== 'string' || this.loading) return null;

    if (this.staticSuggestions.includes(val)) return null;
    if (val.includes('${{')) return null;

    if (val.endsWith('/')) {
      return { notFinal: true };
    }

    if (this.rbacMode) {
      const match = this.rbacSuggestions.find((s) => s.fullId === val);
      if (match) {
        return match.final ? null : { notFinal: true };
      }

      // If not an exact match, check if it's a prefix of any suggestion
      const isPrefix = this.rbacSuggestions.some((s) => s.fullId && s.fullId.startsWith(val + '/'));
      if (isPrefix) {
        return { notFinal: true };
      }

      // Custom path or non-matching path: allow it
      // This enables simulators to test hypothetical resource paths
      return null;
    }

    return null;
  }

  // ControlValueAccessor
  writeValue(value: any): void {
    this.control.setValue(value || '', { emitEvent: false });
  }
  registerOnChange(fn: any): void {
    this.onChange = fn;
  }
  registerOnTouched(fn: any): void {
    this.onTouched = fn;
  }
  setDisabledState(isDisabled: boolean): void {
    this.disabled = isDisabled;
    if (isDisabled) this.control.disable();
    else this.control.enable();
  }
}
