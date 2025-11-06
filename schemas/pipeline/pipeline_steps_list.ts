export default {
  "credential-offer": {
    "$defs": {
      "ActivityOptions": {
        "additionalProperties": false,
        "properties": {
          "retry_policy": {
            "additionalProperties": false,
            "properties": {
              "backoff_coefficient": {
                "type": "number"
              },
              "initial_interval": {
                "type": "string"
              },
              "maximum_attempts": {
                "type": "integer"
              },
              "maximum_interval": {
                "type": "string"
              }
            },
            "type": "object"
          },
          "schedule_to_close_timeout": {
            "type": "string"
          },
          "start_to_close_timeout": {
            "type": "string"
          }
        },
        "type": "object"
      }
    },
    "additionalProperties": false,
    "properties": {
      "activity_options": {
        "$ref": "#/$defs/ActivityOptions"
      },
      "continue_on_error": {
        "type": "boolean"
      },
      "id": {
        "type": "string"
      },
      "metadata": {
        "additionalProperties": true,
        "type": "object"
      },
      "use": {
        "const": "credential-offer",
        "type": "string"
      },
      "with": {
        "properties": {
          "config": {
            "additionalProperties": true,
            "type": "object"
          },
          "credential_id": {
            "type": "string"
          }
        },
        "required": [
          "credential_id"
        ],
        "type": "object"
      }
    },
    "required": [
      "id",
      "use",
      "with"
    ],
    "type": "object"
  },
  "mobile-automation": {
    "$defs": {
      "ActivityOptions": {
        "additionalProperties": false,
        "properties": {
          "retry_policy": {
            "additionalProperties": false,
            "properties": {
              "backoff_coefficient": {
                "type": "number"
              },
              "initial_interval": {
                "type": "string"
              },
              "maximum_attempts": {
                "type": "integer"
              },
              "maximum_interval": {
                "type": "string"
              }
            },
            "type": "object"
          },
          "schedule_to_close_timeout": {
            "type": "string"
          },
          "start_to_close_timeout": {
            "type": "string"
          }
        },
        "type": "object"
      }
    },
    "additionalProperties": false,
    "properties": {
      "activity_options": {
        "$ref": "#/$defs/ActivityOptions"
      },
      "continue_on_error": {
        "type": "boolean"
      },
      "id": {
        "type": "string"
      },
      "metadata": {
        "additionalProperties": true,
        "type": "object"
      },
      "use": {
        "const": "mobile-automation",
        "type": "string"
      },
      "with": {
        "properties": {
          "action_code": {
            "type": "string"
          },
          "action_id": {
            "type": "string"
          },
          "config": {
            "additionalProperties": true,
            "type": "object"
          },
          "parameters": {
            "additionalProperties": {
              "type": "string"
            },
            "type": "object"
          },
          "version_id": {
            "type": "string"
          },
          "video": {
            "type": "boolean"
          }
        },
        "type": "object"
      }
    },
    "required": [
      "id",
      "use",
      "with"
    ],
    "type": "object"
  },
  "use-case-verification-deeplink": {
    "$defs": {
      "ActivityOptions": {
        "additionalProperties": false,
        "properties": {
          "retry_policy": {
            "additionalProperties": false,
            "properties": {
              "backoff_coefficient": {
                "type": "number"
              },
              "initial_interval": {
                "type": "string"
              },
              "maximum_attempts": {
                "type": "integer"
              },
              "maximum_interval": {
                "type": "string"
              }
            },
            "type": "object"
          },
          "schedule_to_close_timeout": {
            "type": "string"
          },
          "start_to_close_timeout": {
            "type": "string"
          }
        },
        "type": "object"
      }
    },
    "additionalProperties": false,
    "properties": {
      "activity_options": {
        "$ref": "#/$defs/ActivityOptions"
      },
      "continue_on_error": {
        "type": "boolean"
      },
      "id": {
        "type": "string"
      },
      "metadata": {
        "additionalProperties": true,
        "type": "object"
      },
      "use": {
        "const": "use-case-verification-deeplink",
        "type": "string"
      },
      "with": {
        "properties": {
          "config": {
            "additionalProperties": true,
            "type": "object"
          },
          "use_case_id": {
            "type": "string"
          }
        },
        "required": [
          "use_case_id"
        ],
        "type": "object"
      }
    },
    "required": [
      "id",
      "use",
      "with"
    ],
    "type": "object"
  }
} as const;
