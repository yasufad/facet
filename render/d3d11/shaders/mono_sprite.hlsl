cbuffer GlobalConstants : register(b0)
{
    float2 u_viewport_size;
    float2 u_atlas_size; // width, height of atlas texture in pixels
};

Texture2D<float> t_atlas : register(t0);
SamplerState s_atlas : register(s0);

struct VSInput
{
    uint vertex_id : SV_VertexID;
    uint instance_id : SV_InstanceID;
    float4 bounds : INST_BOUNDS;           // min_x, min_y, max_x, max_y
    float4 content_mask : INST_MASK;       // min_x, min_y, max_x, max_y
    float4 color : INST_COLOR;             // premultiplied rgba
    float4 tile_bounds : INST_TILE;        // min_x, min_y, max_x, max_y in atlas device pixels
    float4 transform_mat : INST_TRANS_MAT; // rot_scale 2x2: m00, m01, m10, m11
    float2 transform_tx : INST_TRANS_TX;  // translation tx, ty
    float2 pad0 : INST_PAD0;
};

struct PSInput
{
    float4 position : SV_POSITION;
    float2 screen_pos : SCREEN_POS;
    float2 uv : TEXCOORD0;
    float4 content_mask : CONTENT_MASK;
    float4 color : COLOR;
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
    float2 local_pos = float2(
        lerp(input.bounds.x, input.bounds.z, corner.x),
        lerp(input.bounds.y, input.bounds.w, corner.y)
    );

    // Apply transformation
    float2 transformed_pos = float2(
        input.transform_mat.x * local_pos.x + input.transform_mat.y * local_pos.y + input.transform_tx.x,
        input.transform_mat.z * local_pos.x + input.transform_mat.w * local_pos.y + input.transform_tx.y
    );

    float2 ndc = float2(
        (transformed_pos.x / u_viewport_size.x) * 2.0 - 1.0,
        1.0 - (transformed_pos.y / u_viewport_size.y) * 2.0
    );

    float2 uv = float2(
        lerp(input.tile_bounds.x, input.tile_bounds.z, corner.x) / u_atlas_size.x,
        lerp(input.tile_bounds.y, input.tile_bounds.w, corner.y) / u_atlas_size.y
    );

    output.position = float4(ndc, 0.0, 1.0);
    output.screen_pos = transformed_pos;
    output.uv = uv;
    output.content_mask = input.content_mask;
    output.color = input.color;
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

    float coverage = t_atlas.Sample(s_atlas, input.uv);
    return input.color * coverage;
}
