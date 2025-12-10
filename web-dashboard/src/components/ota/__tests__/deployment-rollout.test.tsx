import { render, screen } from "@testing-library/react";
import { DeploymentRollout } from "../deployment-rollout";
import type { Deployment, DeploymentProgress } from "@/lib/ota/types";

const mockDeployment: Deployment = {
  id: "dep-1",
  name: "Blinky v2.1.0 Rollout",
  description: "Phased rollout to all devices",
  firmwareReleaseId: "rel-1",
  firmwareVersion: "2.1.0",
  targetGroups: [
    { name: "Group A", deviceIds: ["dev-1", "dev-2"] },
    { name: "Group B", deviceIds: ["dev-3", "dev-4"] },
  ],
  strategy: "phased",
  status: "running",
  createdAt: "2025-01-01T10:00:00Z",
  createdBy: "admin",
  rolloutConfig: {
    phases: [
      { name: "Phase 1", order: 1, percentage: 25 },
      { name: "Phase 2", order: 2, percentage: 50 },
      { name: "Phase 3", order: 3, percentage: 100 },
    ],
    rollbackOnFailure: true,
  },
};

const mockProgress: DeploymentProgress = {
  deploymentId: "dep-1",
  status: "running",
  totalDevices: 4,
  completedDevices: 1,
  failedDevices: 0,
  pendingDevices: 3,
  currentPhase: "Phase 1",
  phases: [
    {
      phase: { name: "Phase 1", order: 1, percentage: 25 },
      totalDevices: 1,
      completedDevices: 1,
      failedDevices: 0,
      pendingDevices: 0,
      status: "completed",
    },
  ],
};

describe("DeploymentRollout", () => {
  it("renders deployment rollout interface", () => {
    render(
      <DeploymentRollout
        deployment={mockDeployment}
        progress={mockProgress}
        onAction={() => {}}
        actionLoading={null}
      />
    );
    // Check that the main sections are rendered
    expect(screen.getByText("Controls")).toBeInTheDocument();
    expect(screen.getByText("Progress Overview")).toBeInTheDocument();
    expect(screen.getByText("Phases")).toBeInTheDocument();
  });

  it("shows deployment controls", () => {
    render(
      <DeploymentRollout
        deployment={mockDeployment}
        progress={mockProgress}
        onAction={() => {}}
        actionLoading={null}
      />
    );
    // Check that controls section is rendered
    expect(screen.getByText("Controls")).toBeInTheDocument();
  });

  it("displays progress overview", () => {
    render(
      <DeploymentRollout
        deployment={mockDeployment}
        progress={mockProgress}
        onAction={() => {}}
        actionLoading={null}
      />
    );
    expect(screen.getByText("Progress Overview")).toBeInTheDocument();
    expect(screen.getByText("Overall")).toBeInTheDocument();
  });

  it("shows progress information", () => {
    render(
      <DeploymentRollout
        deployment={mockDeployment}
        progress={mockProgress}
        onAction={() => {}}
        actionLoading={null}
      />
    );
    expect(screen.getByText(/1.*4/)).toBeInTheDocument(); // completed/total
  });

  it("displays current phase", () => {
    render(
      <DeploymentRollout
        deployment={mockDeployment}
        progress={mockProgress}
        onAction={() => {}}
        actionLoading={null}
      />
    );
    expect(screen.getByText("Phase 1")).toBeInTheDocument();
    expect(screen.getByText("completed")).toBeInTheDocument();
  });

  it("shows pause button when running", () => {
    render(
      <DeploymentRollout
        deployment={mockDeployment}
        progress={mockProgress}
        onAction={() => {}}
        actionLoading={null}
      />
    );
    const pauseBtn = screen.getByText("Pause");
    expect(pauseBtn).toBeInTheDocument();
  });

  it("shows cancel button", () => {
    render(
      <DeploymentRollout
        deployment={mockDeployment}
        progress={mockProgress}
        onAction={() => {}}
        actionLoading={null}
      />
    );
    const cancelBtn = screen.getByText("Cancel");
    expect(cancelBtn).toBeInTheDocument();
  });
});
