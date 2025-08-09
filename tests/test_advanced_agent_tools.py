from integration.tools import advanced_agent_tools

def test_plan_tasks():
    tasks = advanced_agent_tools.plan_tasks("Build a web app")
    assert isinstance(tasks, list)
    assert tasks

def test_coordinate_agents():
    res = advanced_agent_tools.coordinate_agents(["Task1", "Task2"])
    assert isinstance(res, dict)
    assert res["status"] == "coordinated"