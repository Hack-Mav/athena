"use client";

import type { JsonSchema, JsonSchemaProperty, TemplateConfigFormValues } from "@/lib/templates/types";

interface TemplateConfigFormProps {
  schema: JsonSchema;
  values: TemplateConfigFormValues;
  onChange: (values: TemplateConfigFormValues) => void;
  errors?: Record<string, string>;
}

export function TemplateConfigForm({ schema, values, onChange, errors }: TemplateConfigFormProps) {
  function updateField(key: string, value: unknown) {
    onChange({ ...values, [key]: value });
  }

  function renderField(key: string, prop: JsonSchemaProperty) {
    const value = values[key];
    const stringValue = typeof value === 'string' ? value : String(value ?? '');

    switch (prop.type) {
      case "string":
        if (prop.enum) {
          return (
            <select
              value={stringValue}
              onChange={(e) => updateField(key, e.target.value)}
              className="w-full rounded-md border border-zinc-300 px-3 py-2 text-sm outline-none focus:border-zinc-900 focus:ring-1 focus:ring-zinc-900"
            >
              <option value="">Select...</option>
              {prop.enum.map((opt: string) => (
                <option key={opt} value={opt}>
                  {opt}
                </option>
              ))}
            </select>
          );
        }
        return (
          <input
            type={prop.format === "email" ? "email" : prop.format === "uri" ? "url" : prop.format === "date-time" ? "datetime-local" : "text"}
            value={stringValue}
            onChange={(e) => updateField(key, e.target.value)}
            placeholder={prop.description}
            minLength={prop.minLength}
            maxLength={prop.maxLength}
            pattern={prop.pattern}
            className="w-full rounded-md border border-zinc-300 px-3 py-2 text-sm outline-none focus:border-zinc-900 focus:ring-1 focus:ring-zinc-900"
          />
        );

      case "number":
      case "integer":
        return (
          <input
            type="number"
            value={stringValue}
            onChange={(e) => updateField(key, prop.type === "integer" ? parseInt(e.target.value, 10) : parseFloat(e.target.value))}
            min={prop.minimum}
            max={prop.maximum}
            step={prop.multipleOf ?? (prop.type === "integer" ? 1 : "any")}
            className="w-full rounded-md border border-zinc-300 px-3 py-2 text-sm outline-none focus:border-zinc-900 focus:ring-1 focus:ring-zinc-900"
          />
        );

      case "boolean":
        return (
          <label className="flex items-center gap-2 text-sm text-zinc-700">
            <input
              type="checkbox"
              checked={Boolean(value)}
              onChange={(e) => updateField(key, e.target.checked)}
              className="rounded border-zinc-300 text-zinc-900 focus:ring-zinc-900"
            />
            {prop.title || key}
          </label>
        );

      case "array":
        if (!prop.items) return null;
        return (
          <div>
            {Array.isArray(value) && value.length > 0 ? (
              <ul className="space-y-2">
                {value.map((item, idx) => (
                  <li key={idx} className="flex items-center gap-2">
                    {renderArrayItem(key, idx, prop.items, item)}
                    <button
                      type="button"
                      onClick={() => {
                        const newArr = [...(Array.isArray(value) ? value : [])];
                        newArr.splice(idx, 1);
                        updateField(key, newArr);
                      }}
                      className="text-red-600 hover:text-red-800 text-sm"
                    >
                      Remove
                    </button>
                  </li>
                ))}
              </ul>
            ) : (
              <p className="text-sm text-zinc-600">No items</p>
            )}
            <button
              type="button"
              onClick={() => {
                const newItem = getDefaultForType(prop.items);
                updateField(key, [...(Array.isArray(value) ? value : []), newItem]);
              }}
              className="mt-2 text-sm text-zinc-700 hover:text-zinc-900"
            >
              + Add item
            </button>
          </div>
        );

      case "object":
        if (!prop.properties) return null;
        return (
          <div className="ml-4 border-l-2 border-zinc-200 pl-4 space-y-3">
            {Object.entries(prop.properties).map(([subKey, subProp]) => (
              <div key={subKey}>
                {renderField(`${key}.${subKey}`, subProp)}
              </div>
            ))}
          </div>
        );

      default:
        return null;
    }
  }

  function renderArrayItem(parentKey: string, idx: number, itemSchema: JsonSchemaProperty, value: unknown) {
    const updateItem = (newValue: unknown) => {
      const arr = (values[parentKey] as unknown[]) || [];
      const newArr = [...arr];
      newArr[idx] = newValue;
      updateField(parentKey, newArr);
    };
    
    const stringValue = typeof value === 'string' ? value : String(value ?? '');

    switch (itemSchema.type) {
      case "string":
        if (itemSchema.enum) {
          return (
            <select
              value={stringValue}
              onChange={(e) => updateItem(e.target.value)}
              className="rounded-md border border-zinc-300 px-3 py-2 text-sm outline-none focus:border-zinc-900 focus:ring-1 focus:ring-zinc-900"
            >
              <option value="">Select...</option>
              {itemSchema.enum.map((opt: string) => (
                <option key={opt} value={opt}>
                  {opt}
                </option>
              ))}
            </select>
          );
        }
        return (
          <input
            type="text"
            value={stringValue}
            onChange={(e) => updateItem(e.target.value)}
            className="rounded-md border border-zinc-300 px-3 py-2 text-sm outline-none focus:border-zinc-900 focus:ring-1 focus:ring-zinc-900"
          />
        );

      case "number":
      case "integer":
        return (
          <input
            type="number"
            value={stringValue}
            onChange={(e) => updateItem(itemSchema.type === "integer" ? parseInt(e.target.value, 10) : parseFloat(e.target.value))}
            className="rounded-md border border-zinc-300 px-3 py-2 text-sm outline-none focus:border-zinc-900 focus:ring-1 focus:ring-zinc-900"
          />
        );

      case "boolean":
        return (
          <input
            type="checkbox"
            checked={Boolean(value)}
            onChange={(e) => updateItem(e.target.checked)}
            className="rounded border-zinc-300 text-zinc-900 focus:ring-zinc-900"
          />
        );

      default:
        return null;
    }
  }

  function getDefaultForType(prop: JsonSchemaProperty): unknown {
    if ("default" in prop && prop.default !== undefined) return prop.default;
    switch (prop.type) {
      case "string": return "";
      case "number":
      case "integer": return 0;
      case "boolean": return false;
      case "array": return [];
      case "object": return {};
      default: return null;
    }
  }

  if (!schema.properties) {
    return <p className="text-sm text-zinc-600">No configuration fields.</p>;
  }

  return (
    <form className="space-y-4">
      {Object.entries(schema.properties).map(([key, prop]) => {
        const required = schema.required?.includes(key) ?? false;
        return (
          <div key={key}>
            <label className="block text-sm font-medium text-zinc-700 mb-1">
              {prop.title || key}
              {required && <span className="text-red-500 ml-1">*</span>}
            </label>
            {renderField(key, prop)}
            {prop.description && (
              <p className="mt-1 text-xs text-zinc-500">{prop.description}</p>
            )}
            {errors?.[key] && (
              <p className="mt-1 text-xs text-red-600">{errors[key]}</p>
            )}
          </div>
        );
      })}
    </form>
  );
}
