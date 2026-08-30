cbuffer GlobalConstants : register(b0)
{
    float2 u_viewport_size;
    float2 u_pad;
};

struct VSInput
{
    uint vertex_id : SV_VertexID;
    uint instance_id : SV_InstanceID;
    float4 bounds : INST_BOUNDS;           // min_x, min_y, max_x, max_y
    float4 content_mask : INST_MASK;       // min_x, min_y, max_x, max_y
    float4 color : INST_COLOR;             // premultiplied rgba
    float thickness : INST_THICKNESS;
    uint wavy : INST_WAVY;                 // 0 = straight, 1 = wavy
    float2 pad0 : INST_PAD0;
};

struct PSInput
{
    float4 position : SV_POSITION;
    float2 screen_pos : SCREEN_POS;
    float4 bounds : BOUNDS;
    float4 content_mask : CONTENT_MASK;
    float4 color : COLOR;
    float thickness : THICKNESS;
    uint wavy : WAVY;
};

static const float2 QUAD_CORNERS[6] = {
    float2(0.0, 0.0),
    float2(1.0, 0.0),
    float2(0.0, 1.0),
    float2(0.0, 1.0),
    float2(1.0, 0.0),
    float2(1.0, 1.0)
};

PSInput vs_main(VSInput input)
{
    PSInput output;
    float2 corner = QUAD_CORNERS[input.vertex_id % 6];
    float2 pos = float2(
        lerp(input.bounds.x, input.bounds.z, corner.x),
        lerp(input.bounds.y, input.bounds.w, corner.y)
    );

    float2 ndc = float2(
        (pos.x / u_viewport_size.x) * 2.0 - 1.0,
        1.0 - (pos.y / u_viewport_size.y) * 2.0
    );

    output.position = float4(ndc, 0.0, 1.0);
    output.screen_pos = pos;
    output.bounds = input.bounds;
    output.content_mask = input.content_mask;
    output.color = input.color;
    output.thickness = input.thickness;
    output.wavy = input.wavy;
    return output;
}

float4 ps_main(PSInput input) : SV_TARGET
{
    // Content mask clip
    if (input.content_mask.z > input.content_mask.x && input.content_mask.w > input.content_mask.y)
    {
        if (input.screen_pos.x < input.content_mask.x || input.screen_pos.x > input.content_mask.z ||
            input.screen_pos.y < input.content_mask.y || input.screen_pos.y > input.content_mask.w)
        {
            discard;
        }
    }

    float y_center = (input.bounds.y + input.bounds.w) * 0.5;
    float half_thick = max(0.5, input.thickness * 0.5);

    if (input.wavy != 0)
    {
        float period = max(4.0, input.thickness * 4.0);
        float amplitude = half_thick * 0.8;
        float wave_y = y_center + sin((input.screen_pos.x - input.bounds.x) * (6.2831853 / period)) * amplitude;
        float dist = abs(input.screen_pos.y - wave_y) - half_thick;
        float alpha = saturate(0.5 - dist);
        return input.color * alpha;
    }
    else
    {
        float dist = abs(input.screen_pos.y - y_center) - half_thick;
        float alpha = saturate(0.5 - dist);
        return input.color * alpha;
    }
}
